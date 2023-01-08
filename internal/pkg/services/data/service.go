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

// Find The multipurpose wrapper for collection.Find function.
// With power comes responsibility, check the result yourself, including length, error, etc.
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
	options ...*options.FindOneOptions) Result {
	if reflect.TypeOf(receiver).Kind() != reflect.Ptr {
		return NewErrorResult(errors.New("receiver is not a pointer"))
	}
	findOneResult := collection.FindOne(ctx, filter, options...)
	if findOneResult.Err() != nil {
		// can be mongo.ErrNoDocuments or other type of error
		return NewErrorResult(findOneResult.Err())
	}
	if err := findOneResult.Decode(receiver); err != nil {
		core.Logger.Warnf("Database unmarshal error: %v", err)
		return NewErrorResult(err)
	}
	return NewSuccessResult(findOneResult)

}

func (s *Service) InsertOne(subject any, collection *mongo.Collection, ctx context.Context,
	options ...*options.InsertOneOptions) Result {
	insertOneResult, err := collection.InsertOne(ctx, subject, options...)
	if err != nil {
		return NewErrorResult(err)
	}
	return NewSuccessResult(insertOneResult)
}

// UpdateOne wrapper of UpdateOne function.
// Can be used for Upsert by passing options
func (s *Service) UpdateOne(subject any, collection *mongo.Collection, ctx context.Context,
	filter any, options ...*options.UpdateOptions) Result {
	updateResult, err := collection.UpdateOne(ctx, filter, subject, options...)
	if err != nil {
		core.Logger.Warnf("Database update error: %v", err)
		return NewErrorResult(err)
	}
	return NewSuccessResult(updateResult)
}

func (s *Service) UpdateByID(subject any, id any, collection *mongo.Collection, ctx context.Context, options ...*options.UpdateOptions) Result {
	updateResult, err := collection.UpdateByID(ctx, id, subject, options...)
	if err != nil {
		core.Logger.Warnf("Database update error: %v", err)
		return NewErrorResult(err)
	}
	return NewSuccessResult(updateResult)
}

func (s *Service) DeleteOne(collection *mongo.Collection, ctx context.Context, filter any,
	options ...*options.DeleteOptions) Result {
	deleteResult, err := collection.DeleteOne(ctx, filter, options...)
	if err != nil {
		core.Logger.Warnf("Database delete error: %v", err)
		return NewErrorResult(err)
	}
	return NewSuccessResult(deleteResult)
}

type ServiceConfig struct {
	URI string
}
