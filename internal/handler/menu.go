package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/shubhamjaiswar43/restaurant-management/internal/auth"
	"github.com/shubhamjaiswar43/restaurant-management/internal/helper"
	"github.com/shubhamjaiswar43/restaurant-management/internal/storage/mongodb"
	"github.com/shubhamjaiswar43/restaurant-management/internal/types"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MenuHandler struct {
	MenuStore       *mongodb.MenuStore
	RestaurantStore *mongodb.RestaurantStore
}

func NewMenuHandler(menuStore *mongodb.MenuStore, restaurantStore *mongodb.RestaurantStore) *MenuHandler {
	return &MenuHandler{MenuStore: menuStore, RestaurantStore: restaurantStore}
}

// POST /menu-items
func (h *MenuHandler) CreateMenuItem(w http.ResponseWriter, r *http.Request) {
	slog.Info("CreateMenuItem API called", slog.Time("timestamp", time.Now()))

	claims, ok := r.Context().Value("claims").(*auth.Claims)
	if !ok {
		slog.Error("Missing claims in context")
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Failed to read user info from context")
		return
	}

	slog.Info("CreateMenuItem request received",
		slog.String("user_id", claims.UserID),
		slog.String("role", claims.Role),
		slog.Time("timestamp", time.Now()),
	)

	if claims.Role != "admin" {
		slog.Warn("Unauthorized menu creation attempt", slog.String("user_id", claims.UserID))
		helper.WriteSimpleError(w, http.StatusForbidden, "Only admin can create menu items")
		return
	}

	var item types.MenuItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		slog.Error("Invalid JSON body", slog.String("error", err.Error()))
		helper.WriteSimpleError(w, http.StatusBadRequest, "Invalid JSON body: "+err.Error())
		return
	}

	if err := helper.ValidateStruct(item); err != nil {
		slog.Warn("Menu item validation failed", slog.String("error", err.Error()))
		helper.WriteValidationError(w, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if the restaurant exists
	restaurant, err := h.RestaurantStore.GetByID(ctx, item.Restaurant.Hex())
	if err != nil {
		slog.Error("Failed to check restaurant existence", slog.String("error", err.Error()))
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Error checking restaurant: "+err.Error())
		return
	}
	if restaurant == nil {
		slog.Warn("Menu creation failed: restaurant not found", slog.String("restaurant_id", item.Restaurant.Hex()))
		helper.WriteSimpleError(w, http.StatusBadRequest, "Restaurant does not exist")
		return
	}

	// Check if item already exists in this restaurant
	exists, err := h.MenuStore.GetByNameAndRestaurant(ctx, item.Name, item.Restaurant)
	if err != nil {
		slog.Error("Failed to check menu item existence", slog.String("error", err.Error()))
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Error checking existing menu item: "+err.Error())
		return
	}
	if exists != nil {
		slog.Warn("Duplicate menu item creation attempt",
			slog.String("menu_name", item.Name),
			slog.String("restaurant_id", item.Restaurant.Hex()),
		)
		helper.WriteSimpleError(w, http.StatusConflict, "Menu item with this name already exists in this restaurant")
		return
	}

	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()

	created, err := h.MenuStore.CreateMenuItem(ctx, &item)
	if err != nil {
		slog.Error("Failed to create menu item in DB", slog.String("error", err.Error()))
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Failed to create menu item: "+err.Error())
		return
	}

	slog.Info("Menu item created successfully",
		slog.String("menu_id", created.ID.Hex()),
		slog.String("menu_name", created.Name),
		slog.String("restaurant_id", created.Restaurant.Hex()),
		slog.String("created_by", claims.UserID),
		slog.Time("timestamp", time.Now()),
	)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"message":       "Menu item created successfully",
		"menu_item":     created,
		"restaurant_id": created.Restaurant.Hex(),
		"created_by":    claims.UserID,
		"created_at":    time.Now().Format(time.RFC3339),
	})
}

// GET /menu-items?restaurant_id=<id>
func (h *MenuHandler) GetMenuItems(w http.ResponseWriter, r *http.Request) {
	slog.Info("GetMenuItems API called", slog.Time("timestamp", time.Now()))

	claims, ok := r.Context().Value("claims").(*auth.Claims)
	if !ok {
		slog.Error("Missing claims in context")
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Failed to read user info from context")
		return
	}

	slog.Info("GetMenuItems request received",
		slog.String("user_id", claims.UserID),
		slog.String("role", claims.Role),
		slog.Time("timestamp", time.Now()),
	)

	restaurantIDStr := r.URL.Query().Get("restaurant_id")
	if restaurantIDStr == "" {
		slog.Warn("Missing restaurant_id in query params")
		helper.WriteSimpleError(w, http.StatusBadRequest, "Missing required query parameter: restaurant_id")
		return
	}

	_, err := primitive.ObjectIDFromHex(restaurantIDStr)
	if err != nil {
		slog.Warn("Invalid restaurant_id format", slog.String("restaurant_id", restaurantIDStr))
		helper.WriteSimpleError(w, http.StatusBadRequest, "Invalid restaurant_id format")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	items, err := h.MenuStore.GetByRestaurant(ctx, restaurantIDStr)
	if err != nil {
		slog.Error("Failed to fetch menu items", slog.String("error", err.Error()))
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Failed to fetch menu items: "+err.Error())
		return
	}

	slog.Info("Menu items fetched successfully",
		slog.Int("count", len(items)),
		slog.String("restaurant_id", restaurantIDStr),
		slog.String("requested_by", claims.UserID),
		slog.Time("timestamp", time.Now()),
	)

	json.NewEncoder(w).Encode(map[string]any{
		"count":         len(items),
		"menu_items":    items,
		"restaurant_id": restaurantIDStr,
		"requested_by":  claims.UserID,
	})
}
