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
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OrderHandler struct {
	Store *mongodb.OrderStore
}

func NewOrderHandler(store *mongodb.OrderStore) *OrderHandler {
	return &OrderHandler{Store: store}
}

// POST /orders
func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	slog.Info("CreateOrder API called", slog.Time("timestamp", time.Now()))

	claims, ok := r.Context().Value("claims").(*auth.Claims)
	if !ok {
		slog.Error("Missing claims in context")
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Failed to read user info from context")
		return
	}

	var order types.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		slog.Error("Invalid JSON body", slog.String("error", err.Error()))
		helper.WriteSimpleError(w, http.StatusBadRequest, "Invalid JSON body: "+err.Error())
		return
	}

	if err := helper.ValidateStructExcept(order, "UserID"); err != nil {
		slog.Warn("Order validation failed", slog.String("error", err.Error()))
		helper.WriteValidationError(w, err)
		return
	}

	if claims.Role == "admin" {
		if order.UserID.IsZero() {
			slog.Warn("Admin missing user_id in order creation")
			helper.WriteSimpleError(w, http.StatusBadRequest, "Admin must specify user_id for the order")
			return
		}
	} else {
		userObjID, err := primitive.ObjectIDFromHex(claims.UserID)
		if err != nil {
			slog.Error("Invalid user_id in token", slog.String("error", err.Error()))
			helper.WriteSimpleError(w, http.StatusUnauthorized, "Invalid user ID in token: "+err.Error())
			return
		}
		order.UserID = userObjID
	}

	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	created, err := h.Store.CreateOrder(ctx, &order)
	if err != nil {
		slog.Error("Failed to create order", slog.String("error", err.Error()))
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Failed to create order: "+err.Error())
		return
	}

	slog.Info("Order created successfully",
		slog.String("order_id", created.ID.Hex()),
		slog.String("user_id", claims.UserID),
		slog.Time("timestamp", time.Now()),
	)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"message":   "Order created successfully",
		"order":     created,
		"createdAt": time.Now().Format(time.RFC3339),
	})
}

// GET /orders - only admin
func (h *OrderHandler) GetAllOrders(w http.ResponseWriter, r *http.Request) {
	slog.Info("GetAllOrders API called", slog.Time("timestamp", time.Now()))

	claims, ok := r.Context().Value("claims").(*auth.Claims)
	if !ok {
		slog.Error("Missing claims in context")
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Failed to read user info from context")
		return
	}

	if claims.Role != "admin" {
		slog.Warn("Unauthorized access attempt to GetAllOrders", slog.String("user_id", claims.UserID))
		helper.WriteSimpleError(w, http.StatusForbidden, "Only admin can view all orders")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orders, err := h.Store.GetAllOrders(ctx)
	if err != nil {
		slog.Error("Failed to fetch orders", slog.String("error", err.Error()))
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Failed to fetch orders: "+err.Error())
		return
	}

	slog.Info("Fetched all orders successfully",
		slog.Int("count", len(orders)),
		slog.String("requested_by", claims.UserID),
		slog.Time("timestamp", time.Now()),
	)

	json.NewEncoder(w).Encode(map[string]any{
		"count":   len(orders),
		"orders":  orders,
		"fetched": time.Now().Format(time.RFC3339),
	})
}

// GET /orders/{id}
func (h *OrderHandler) GetOrderByID(w http.ResponseWriter, r *http.Request) {
	slog.Info("GetOrderByID API called", slog.Time("timestamp", time.Now()))

	claims, ok := r.Context().Value("claims").(*auth.Claims)
	if !ok {
		slog.Error("Missing claims in context")
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Failed to read user info from context")
		return
	}

	idStr := r.URL.Path[len("/orders/"):]
	orderID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		slog.Warn("Invalid order ID", slog.String("id", idStr))
		helper.WriteSimpleError(w, http.StatusBadRequest, "Invalid order ID format")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	order, err := h.Store.GetOrderByID(ctx, orderID)
	if err != nil {
		slog.Error("Failed to fetch order", slog.String("error", err.Error()))
		helper.WriteSimpleError(w, http.StatusInternalServerError, "Failed to fetch order: "+err.Error())
		return
	}
	if order == nil {
		slog.Warn("Order not found", slog.String("order_id", idStr))
		helper.WriteSimpleError(w, http.StatusNotFound, "Order not found")
		return
	}

	if claims.Role != "admin" && order.UserID.Hex() != claims.UserID {
		slog.Warn("Forbidden order access", slog.String("order_id", order.ID.Hex()), slog.String("user_id", claims.UserID))
		helper.WriteSimpleError(w, http.StatusForbidden, "You can only access your own orders")
		return
	}

	slog.Info("Fetched order successfully",
		slog.String("order_id", order.ID.Hex()),
		slog.String("requested_by", claims.UserID),
		slog.Time("timestamp", time.Now()),
	)

	json.NewEncoder(w).Encode(map[string]any{
		"order":       order,
		"requestedBy": claims.UserID,
		"fetchedAt":   time.Now().Format(time.RFC3339),
	})
}
