package clients

import (
	"go.mongodb.org/mongo-driver/mongo"
)

var MongoClient *mongo.Client

func RegisterMongoClient(client *mongo.Client) {
	MongoClient = client
}
