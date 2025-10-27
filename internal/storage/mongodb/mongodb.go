package mongodb

import (
	"context"
	"time"

	"github.com/shubhamjaiswar43/restify/internal/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDb struct {
	Db *mongo.Database
}

func New(cfg *config.Config) (*MongoDb, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.StoragePath))
	if err != nil {
		return &MongoDb{}, err
	}
	db := client.Database(cfg.DatabaseName)
	return &MongoDb{
		Db: db,
	}, nil
}
