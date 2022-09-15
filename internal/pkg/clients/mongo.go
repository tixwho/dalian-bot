package clients

import (
	"go.mongodb.org/mongo-driver/mongo"
)

var MongoClient *mongo.Client
var DalianDB *mongo.Database

func RegisterMongoClient(client *mongo.Client) {
	MongoClient = client
}

func ConnectToDB(dbName string) {
	DalianDB = MongoClient.Database(dbName)
}
