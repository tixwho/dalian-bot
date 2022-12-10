package data

import (
	"dalian-bot/internal/pkg/clients"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetCollection(name string, opts ...*options.CollectionOptions) *mongo.Collection {
	return clients.DalianDB.Collection(name, opts...)
}
