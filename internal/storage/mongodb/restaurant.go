package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shubhamjaiswar43/restify/internal/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RestaurantStore struct {
	Collection *mongo.Collection
}

func NewRestaurantStore(collection *mongo.Collection) *RestaurantStore {
	return &RestaurantStore{Collection: collection}
}

// Insert a new restaurant
func (s *RestaurantStore) CreateRestaurant(ctx context.Context, r *types.Restaurant) (*types.Restaurant, error) {
	now := time.Now()
	r.CreatedAt = now
	r.UpdatedAt = now
	res, err := s.Collection.InsertOne(ctx, r)
	if err != nil {
		return nil, err
	}
	r.ID = res.InsertedID.(primitive.ObjectID)
	return r, nil
}

// GetByName finds a restaurant by name
func (s *RestaurantStore) GetByName(ctx context.Context, name string) (*types.Restaurant, error) {
	filter := bson.M{"name": name}
	var restaurant types.Restaurant
	err := s.Collection.FindOne(ctx, filter).Decode(&restaurant)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil // not found
		}
		return nil, err
	}
	return &restaurant, nil
}

// GetById finds a restaurant by name
func (s *RestaurantStore) GetByID(ctx context.Context, id string) (*types.Restaurant, error) {
	objId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid restaurant ID: %v", err)
	}
	filter := bson.M{"_id": objId}
	var restaurant types.Restaurant
	err = s.Collection.FindOne(ctx, filter).Decode(&restaurant)
	fmt.Print(id, restaurant, err)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil // not found
		}
		return nil, err
	}
	return &restaurant, nil
}

// Get all restaurants
func (s *RestaurantStore) GetAllRestaurants(ctx context.Context) ([]*types.Restaurant, error) {
	cursor, err := s.Collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var restaurants []*types.Restaurant
	for cursor.Next(ctx) {
		var r types.Restaurant
		if err := cursor.Decode(&r); err != nil {
			return nil, err
		}
		restaurants = append(restaurants, &r)
	}
	return restaurants, nil
}
