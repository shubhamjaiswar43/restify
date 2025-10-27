package mongodb

import (
	"context"
	"errors"
	"fmt"

	"github.com/shubhamjaiswar43/restaurant-management/internal/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// UserStore defines MongoDB operations for users.
type UserStore struct {
	Collection *mongo.Collection
}

// NewUserStore initializes a new UserStore.
func NewUserStore(collection *mongo.Collection) *UserStore {
	return &UserStore{
		Collection: collection,
	}
}

// CreateUser inserts a new user document.
func (s *UserStore) CreateUser(ctx context.Context, u *types.User) error {
	_, err := s.Collection.InsertOne(ctx, u)
	return err
}

// GetUserByEmail retrieves a user by email address.
func (s *UserStore) GetUserByEmail(ctx context.Context, email string) (*types.User, error) {
	var user types.User
	err := s.Collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByID retrieves a user by their MongoDB ObjectID.
func (s *UserStore) GetUserByID(ctx context.Context, id string) (*types.User, error) {
	var user types.User
	objId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid restaurant ID: %v", err)
	}
	err = s.Collection.FindOne(ctx, bson.M{"_id": objId}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetAllUsers returns all users (useful for admin panel).
func (s *UserStore) GetAllUsers(ctx context.Context) ([]types.User, error) {
	cursor, err := s.Collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []types.User
	for cursor.Next(ctx) {
		var u types.User
		if err := cursor.Decode(&u); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, cursor.Err()
}

// DeleteUser removes a user by ID (admin operation).
func (s *UserStore) DeleteUser(ctx context.Context, id string) error {
	result, err := s.Collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}
