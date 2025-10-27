package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"log/slog"

	"github.com/shubhamjaiswar43/restaurant-management/internal/auth"
	"github.com/shubhamjaiswar43/restaurant-management/internal/helper"
	"github.com/shubhamjaiswar43/restaurant-management/internal/storage/mongodb"
	"github.com/shubhamjaiswar43/restaurant-management/internal/types"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	Store       *mongodb.UserStore
	JWT         *auth.JWTManager
	AdminSecret string
}

func NewUserHandler(store *mongodb.UserStore, jwt *auth.JWTManager, adminSecret string) *UserHandler {
	return &UserHandler{
		Store:       store,
		JWT:         jwt,
		AdminSecret: adminSecret,
	}
}

// POST /signup
func (h *UserHandler) Signup(w http.ResponseWriter, r *http.Request) {
	slog.Info("Signup API called", slog.Time("timestamp", time.Now()))

	var user types.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		slog.Error("Invalid JSON body", slog.String("error", err.Error()))
		helper.WriteSimpleError(w, http.StatusBadRequest, "Invalid JSON body: "+err.Error())
		return
	}

	if err := helper.ValidateStruct(user); err != nil {
		slog.Warn("User validation failed", slog.String("error", err.Error()))
		helper.WriteValidationError(w, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	existing, err := h.Store.GetUserByEmail(ctx, user.Email)
	if err != nil {
		slog.Error("Failed to check user existence", slog.String("error", err.Error()))
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Database error while checking user: "+err.Error())
		return
	}
	if existing != nil {
		slog.Warn("User tried to register with existing email", slog.String("email", user.Email))
		helper.WriteSimpleError(w, http.StatusConflict, "Email already registered")
		return
	}

	if user.Role == "admin" {
		adminKey := r.Header.Get("Admin-Secret")
		if adminKey == "" {
			slog.Warn("Missing Admin-Secret header for admin creation")
			helper.WriteSimpleError(w, http.StatusUnauthorized, "Missing Admin-Secret header")
			return
		}
		if adminKey != h.AdminSecret {
			slog.Warn("Invalid Admin-Secret key used for admin signup")
			helper.WriteSimpleError(w, http.StatusUnauthorized, "Invalid Admin-Secret key")
			return
		}
	}

	if user.Role == "" {
		user.Role = "customer"
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("Failed to hash password", slog.String("error", err.Error()))
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Failed to hash password: "+err.Error())
		return
	}
	user.Password = string(hashed)

	user.ID = primitive.NewObjectID()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	if err := h.Store.CreateUser(ctx, &user); err != nil {
		slog.Error("Failed to create user in DB", slog.String("error", err.Error()))
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Failed to create user: "+err.Error())
		return
	}

	slog.Info("User created successfully",
		slog.String("user_id", user.ID.Hex()),
		slog.String("email", user.Email),
		slog.String("role", user.Role),
		slog.Time("timestamp", time.Now()),
	)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User created successfully",
		"user_id": user.ID.Hex(),
	})
}

// POST /login
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	slog.Info("Login API called", slog.Time("timestamp", time.Now()))

	var req struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Invalid JSON body", slog.String("error", err.Error()))
		helper.WriteSimpleError(w, http.StatusBadRequest, "Invalid JSON body: "+err.Error())
		return
	}

	if err := helper.ValidateStruct(req); err != nil {
		slog.Warn("Login validation failed", slog.String("error", err.Error()))
		helper.WriteValidationError(w, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := h.Store.GetUserByEmail(ctx, req.Email)
	if err != nil {
		slog.Error("Failed to fetch user from DB", slog.String("error", err.Error()))
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if user == nil {
		slog.Warn("Invalid login attempt", slog.String("email", req.Email))
		helper.WriteSimpleError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		slog.Warn("Invalid password", slog.String("email", req.Email))
		helper.WriteSimpleError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	token, err := h.JWT.Generate(user.ID.Hex(), user.Role)
	if err != nil {
		slog.Error("Failed to generate JWT", slog.String("error", err.Error()))
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Failed to generate token: "+err.Error())
		return
	}

	slog.Info("User logged in successfully",
		slog.String("user_id", user.ID.Hex()),
		slog.String("email", user.Email),
		slog.Time("timestamp", time.Now()),
	)

	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
		"role":  user.Role,
	})
}

// GET /users (admin only)
func (h *UserHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	slog.Info("GetAllUsers API called", slog.Time("timestamp", time.Now()))

	claims, ok := r.Context().Value("claims").(*auth.Claims)
	if !ok {
		slog.Error("Failed to read claims from context")
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Failed to read user info from context")
		return
	}

	if claims.Role != "admin" {
		slog.Warn("Unauthorized access attempt", slog.String("user_id", claims.UserID))
		helper.WriteSimpleError(w, http.StatusForbidden, "Unauthorized access — admin only")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	users, err := h.Store.GetAllUsers(ctx)
	if err != nil {
		slog.Error("Failed to fetch users", slog.String("error", err.Error()))
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Failed to fetch users: "+err.Error())
		return
	}

	slog.Info("Fetched all users successfully",
		slog.String("requested_by", claims.UserID),
		slog.Int("count", len(users)),
		slog.Time("timestamp", time.Now()),
	)

	json.NewEncoder(w).Encode(users)
}
