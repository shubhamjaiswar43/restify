package types

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Base model for common fields
type Base struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// Restaurant entity
type Restaurant struct {
	Base        `bson:",inline"`
	Name        string   `bson:"name" json:"name" validate:"required,min=1,max=100"`
	Address     string   `bson:"address" json:"address" validate:"required,min=1,max=200"`
	Phone       string   `bson:"phone" json:"phone" validate:"required,e164"` // e164 pattern for phone
	Description string   `bson:"description,omitempty" json:"description,omitempty" validate:"max=500"`
	MenuItems   []string `bson:"menu_items,omitempty" json:"menu_items,omitempty"`
	IsActive    bool     `bson:"is_active" json:"is_active"`
}

// MenuItem entity
type MenuItem struct {
	Base       `bson:",inline"`
	Restaurant primitive.ObjectID `bson:"restaurant_id" json:"restaurant_id" validate:"required"`
	Name       string             `bson:"name" json:"name" validate:"required,min=1,max=100"`
	Category   string             `bson:"category" json:"category" validate:"required,min=1,max=50"`
	Price      float64            `bson:"price" json:"price" validate:"required,gt=0"`
	Available  bool               `bson:"available" json:"available"`
}

// Order entity
type Order struct {
	Base       `bson:",inline"`
	UserID     primitive.ObjectID `bson:"user_id" json:"user_id" validate:"required"`
	Restaurant primitive.ObjectID `bson:"restaurant_id" json:"restaurant_id" validate:"required"`
	Items      []OrderItem        `bson:"items" json:"items" validate:"required,min=1,dive"` // at least 1 item
	Status     string             `bson:"status" json:"status" validate:"required,oneof=pending preparing ready completed cancelled"`
	TotalPrice float64            `bson:"total_price" json:"total_price" validate:"required,gte=0"`
}

// OrderItem sub-document
type OrderItem struct {
	MenuItemID primitive.ObjectID `bson:"menu_item_id" json:"menu_item_id" validate:"required"`
	Quantity   int                `bson:"quantity" json:"quantity" validate:"required,gt=0"`
	Price      float64            `bson:"price" json:"price" validate:"required,gt=0"`
}
