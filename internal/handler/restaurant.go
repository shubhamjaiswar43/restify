package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"log/slog"

	"github.com/shubhamjaiswar43/restify/internal/auth"
	"github.com/shubhamjaiswar43/restify/internal/helper"
	"github.com/shubhamjaiswar43/restify/internal/storage/mongodb"
	"github.com/shubhamjaiswar43/restify/internal/types"
)

type RestaurantHandler struct {
	Store *mongodb.RestaurantStore
}

func NewRestaurantHandler(store *mongodb.RestaurantStore) *RestaurantHandler {
	return &RestaurantHandler{Store: store}
}

// POST /restaurants
func (h *RestaurantHandler) CreateRestaurant(w http.ResponseWriter, r *http.Request) {
	slog.Info("CreateRestaurant API called", slog.Time("timestamp", time.Now()))

	claims, ok := r.Context().Value("claims").(*auth.Claims)
	if !ok {
		slog.Error("Missing claims in context")
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Failed to read user info from context")
		return
	}

	slog.Info("CreateRestaurant request received",
		slog.String("user_id", claims.UserID),
		slog.String("role", claims.Role),
		slog.Time("timestamp", time.Now()),
	)

	var restaurant types.Restaurant
	if err := json.NewDecoder(r.Body).Decode(&restaurant); err != nil {
		slog.Error("Invalid JSON body", slog.String("error", err.Error()))
		helper.WriteSimpleError(w, http.StatusBadRequest, "Invalid JSON body: "+err.Error())
		return
	}

	if restaurant.Name == "" && restaurant.Address == "" && restaurant.Phone == "" {
		slog.Warn("Empty request body for restaurant creation")
		helper.WriteSimpleError(w, http.StatusBadRequest, "Request body cannot be empty. Add valid name, address, and phone")
		return
	}

	if err := helper.ValidateStruct(restaurant); err != nil {
		slog.Warn("Restaurant validation failed", slog.String("error", err.Error()))
		helper.WriteValidationError(w, err)
		return
	}

	restaurant.IsActive = true
	restaurant.CreatedAt = time.Now()
	restaurant.UpdatedAt = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if a restaurant with the same name already exists
	exists, err := h.Store.GetByName(ctx, restaurant.Name)
	if err != nil {
		slog.Error("Failed to check restaurant existence", slog.String("error", err.Error()))
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Error checking existing restaurant: "+err.Error())
		return
	}
	if exists != nil {
		slog.Warn("Duplicate restaurant creation attempt", slog.String("restaurant_name", restaurant.Name))
		helper.WriteSimpleError(w, http.StatusConflict, "Restaurant with this name already exists")
		return
	}

	created, err := h.Store.CreateRestaurant(ctx, &restaurant)
	if err != nil {
		slog.Error("Failed to create restaurant in DB", slog.String("error", err.Error()))
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Failed to create restaurant: "+err.Error())
		return
	}

	slog.Info("Restaurant created successfully",
		slog.String("restaurant_id", created.ID.Hex()),
		slog.String("restaurant_name", created.Name),
		slog.String("created_by", claims.UserID),
		slog.Time("timestamp", time.Now()),
	)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"message":    "Restaurant created successfully",
		"restaurant": created,
		"created_by": claims.UserID,
		"created_at": time.Now().Format(time.RFC3339),
	})
}

// GET /restaurants
func (h *RestaurantHandler) GetRestaurants(w http.ResponseWriter, r *http.Request) {
	slog.Info("GetRestaurants API called", slog.Time("timestamp", time.Now()))

	claims, ok := r.Context().Value("claims").(*auth.Claims)
	if !ok {
		slog.Error("Missing claims in context")
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Failed to read user info from context")
		return
	}

	slog.Info("GetRestaurants request received",
		slog.String("user_id", claims.UserID),
		slog.String("role", claims.Role),
		slog.Time("timestamp", time.Now()),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	restaurants, err := h.Store.GetAllRestaurants(ctx)
	if err != nil {
		slog.Error("Failed to fetch restaurants", slog.String("error", err.Error()))
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Failed to fetch restaurants: "+err.Error())
		return
	}

	slog.Info("Fetched all restaurants successfully",
		slog.Int("count", len(restaurants)),
		slog.String("requested_by", claims.UserID),
		slog.Time("timestamp", time.Now()),
	)
	json.NewEncoder(w).Encode(map[string]any{
		"count":        len(restaurants),
		"restaurants":  restaurants,
		"requested_by": claims.UserID,
	})
}
