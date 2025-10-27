package mongodb

import (
	"context"
	"errors"

	"github.com/shubhamjaiswar43/restify/internal/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MenuStore struct {
	Collection *mongo.Collection
}

func NewMenuStore(collection *mongo.Collection) *MenuStore {
	return &MenuStore{Collection: collection}
}

func (s *MenuStore) CreateMenuItem(ctx context.Context, item *types.MenuItem) (*types.MenuItem, error) {
	result, err := s.Collection.InsertOne(ctx, item)
	if err != nil {
		return nil, err
	}
	item.ID = result.InsertedID.(primitive.ObjectID)
	return item, nil
}

func (s *MenuStore) GetByNameAndRestaurant(ctx context.Context, name string, restaurantID primitive.ObjectID) (*types.MenuItem, error) {
	filter := bson.M{"name": name, "restaurant_id": restaurantID}
	var item types.MenuItem
	err := s.Collection.FindOne(ctx, filter).Decode(&item)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (s *MenuStore) GetByRestaurant(ctx context.Context, restaurantID string) ([]*types.MenuItem, error) {
	filter := bson.M{}
	if restaurantID != "" {
		id, err := primitive.ObjectIDFromHex(restaurantID)
		if err != nil {
			return nil, err
		}
		filter["restaurant_id"] = id
	}

	cursor, err := s.Collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []*types.MenuItem
	for cursor.Next(ctx) {
		var m types.MenuItem
		if err := cursor.Decode(&m); err != nil {
			return nil, err
		}
		items = append(items, &m)
	}
	return items, nil
}
