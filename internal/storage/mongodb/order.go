package mongodb

import (
	"context"
	"errors"
	"time"

	"github.com/shubhamjaiswar43/restaurant-management/internal/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type OrderStore struct {
	Collection *mongo.Collection
}

func NewOrderStore(collection *mongo.Collection) *OrderStore {
	return &OrderStore{Collection: collection}
}

// CreateOrder inserts a new order
func (s *OrderStore) CreateOrder(ctx context.Context, order *types.Order) (*types.Order, error) {
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()
	res, err := s.Collection.InsertOne(ctx, order)
	if err != nil {
		return nil, err
	}
	order.ID = res.InsertedID.(primitive.ObjectID)
	return order, nil
}

// GetAllOrders - for admin
func (s *OrderStore) GetAllOrders(ctx context.Context) ([]*types.Order, error) {
	cursor, err := s.Collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var orders []*types.Order
	for cursor.Next(ctx) {
		var o types.Order
		if err := cursor.Decode(&o); err != nil {
			return nil, err
		}
		orders = append(orders, &o)
	}
	return orders, nil
}

// GetOrderByID fetches a single order by ID
func (s *OrderStore) GetOrderByID(ctx context.Context, id primitive.ObjectID) (*types.Order, error) {
	var order types.Order
	err := s.Collection.FindOne(ctx, bson.M{"_id": id}).Decode(&order)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}
