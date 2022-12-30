package data

import (
	"dalian-bot/internal/pkg/core"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/context"
	"reflect"
	"sync"
)

type Service struct {
	ServiceConfig
	Client *mongo.Client
}

func (s *Service) Name() string {
	return "data"
}

func (s *Service) Init(reg *core.ServiceRegistry) error {
	reg.RegisterService(s)
	return nil
}

func (s *Service) Start(wg *sync.WaitGroup) {
	mongoClient, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(s.URI))
	if err != nil {
		fmt.Println("failed opening Mongo connection.")
		panic(err)
	}
	s.Client = mongoClient
	core.Logger.Debugf("Service [%s] is now online.", reflect.TypeOf(s))
	wg.Done()
}

func (s *Service) Stop(wg *sync.WaitGroup) error {
	if err := s.Client.Disconnect(context.TODO()); err != nil {
		fmt.Println("error closing Mongo connection!")
	}
	core.Logger.Debugf("Service [%s] is successfully closed.", reflect.TypeOf(s))
	wg.Done()
	return nil
}

func (s *Service) Status() error {
	//TODO implement me
	panic("implement me")
}

func (s *Service) GetCollection(name string, opts ...*options.CollectionOptions) *mongo.Collection {
	return s.Client.Database("dalian").Collection(name, opts...)
}

func (s *Service) Find(receiver any, collection *mongo.Collection, ctx context.Context, filter any, options ...*options.FindOptions) error {
	if reflect.TypeOf(receiver).Kind() != reflect.Ptr {
		return errors.New("receiver is not a pointer")
	}
	findCursor, err := collection.Find(ctx, filter, options...)
	if err != nil {
		core.Logger.Warnf("Database cursor error: %v", err)
		return err
	}
	if err = findCursor.All(context.TODO(), receiver); err != nil {
		core.Logger.Warnf("Database unmarshal error: %v", err)
		return err
	}
	return nil
}

func (s *Service) FindOne(receiver any, collection *mongo.Collection, ctx context.Context, filter any,
	options ...*options.FindOneOptions) error {
	if reflect.TypeOf(receiver).Kind() != reflect.Ptr {
		return errors.New("receiver is not a pointer")
	}
	findOneResult := collection.FindOne(ctx, filter, options...)
	if findOneResult.Err() != nil {
		// can be mongo.ErrNoDocuments or other type of error
		return findOneResult.Err()
	}
	if err := findOneResult.Decode(receiver); err != nil {
		core.Logger.Warnf("Database unmarshal error: %v", err)
		return err
	}
	return nil

}

func (s *Service) InsertOne(subject any, collection *mongo.Collection, ctx context.Context, options ...*options.InsertOneOptions) error {
	_, err := collection.InsertOne(ctx, subject, options...)
	if err != nil {
		return err
	}
	return nil
}

type ServiceConfig struct {
	URI string
}

func GetCollection(name string, opts ...*options.CollectionOptions) *mongo.Collection {
	return DalianDB.Collection(name, opts...)
}

var MongoClient *mongo.Client
var DalianDB *mongo.Database

func RegisterMongoClient(client *mongo.Client) {
	MongoClient = client
}

func ConnectToDB(dbName string) {
	DalianDB = MongoClient.Database(dbName)
}
