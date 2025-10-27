package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shubhamjaiswar43/restify/internal/auth"
	"github.com/shubhamjaiswar43/restify/internal/config"
	"github.com/shubhamjaiswar43/restify/internal/handler"
	"github.com/shubhamjaiswar43/restify/internal/storage/mongodb"
)

func rootMessage(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Welcome to Restaurant Management System"))
}

func main() {
	// Load config
	cfg := config.MustLoad()
	// MongoDB setup
	dbClient, err := mongodb.New(cfg)
	if err != nil {
		slog.Error("DB connection failed", slog.String("error", err.Error()))
		return
	}

	// Initialize stores
	userStore := mongodb.NewUserStore(dbClient.Db.Collection("users"))
	restaurantStore := mongodb.NewRestaurantStore(dbClient.Db.Collection("restaurants"))
	menuStore := mongodb.NewMenuStore(dbClient.Db.Collection("menu"))
	orderStore := mongodb.NewOrderStore(dbClient.Db.Collection("orders"))

	// Initialize handlers
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, 24*time.Hour)
	userHandler := handler.NewUserHandler(userStore, jwtManager, cfg.AdminSecret)
	restaurantHandler := handler.NewRestaurantHandler(restaurantStore)
	menuHandler := handler.NewMenuHandler(menuStore, restaurantStore)
	orderHandler := handler.NewOrderHandler(orderStore)

	// middlewares
	authMiddleware := auth.NewAuthMiddleware(cfg.JWTSecret)

	// Router
	router := http.NewServeMux()
	router.HandleFunc("/", rootMessage)

	// Auth routes
	router.HandleFunc("/signup", userHandler.Signup)
	router.HandleFunc("/login", userHandler.Login)

	// Restaurant routes with role-based JWT middleware
	router.HandleFunc("/restaurants", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			authMiddleware("admin", "customer")(restaurantHandler.GetRestaurants)(w, r)
		case http.MethodPost:
			authMiddleware("admin")(restaurantHandler.CreateRestaurant)(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("Method Not Allowed"))
			slog.Warn("method not allowed", slog.String("method", r.Method), slog.String("path", r.URL.Path))
		}
	})

	// Menu routes
	router.HandleFunc("/menu-items", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			authMiddleware("admin", "customer")(menuHandler.GetMenuItems)(w, r)
		} else if r.Method == http.MethodPost {
			authMiddleware("admin")(menuHandler.CreateMenuItem)(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Order routes
	router.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			authMiddleware("admin", "customer")(orderHandler.CreateOrder)(w, r)
		case http.MethodGet:
			authMiddleware("admin")(orderHandler.GetAllOrders)(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	router.HandleFunc("/orders/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			authMiddleware("admin", "customer")(orderHandler.GetOrderByID)(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// HTTP server setup
	server := http.Server{
		Handler: router,
		Addr:    cfg.Addr,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("Server running", slog.String("host", "http://"+cfg.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", slog.String("error", err.Error()))
		}
	}()

	<-done
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server shutdown failed", slog.String("error", err.Error()))
	} else {
		slog.Info("Server shutdown successfully")
	}
}
