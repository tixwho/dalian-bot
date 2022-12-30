package data

import (
	"dalian-bot/internal/pkg/core"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Result struct {
	err    error
	result any
}

func (r Result) Err() error {
	return r.err
}

func (r Result) SingleResult() *mongo.SingleResult {
	if r.result == nil {
		return nil
	}
	return r.result.(*mongo.SingleResult)
}

func (r Result) InsertOneResult() *mongo.InsertOneResult {
	if r.result == nil {
		return nil
	}
	return r.result.(*mongo.InsertOneResult)
}

func (r Result) UpdateResult() *mongo.UpdateResult {
	if r.result == nil {
		return nil
	}
	return r.result.(*mongo.UpdateResult)
}

func (r Result) DeleteResult() *mongo.DeleteResult {
	if r.result == nil {
		return nil
	}
	return r.result.(*mongo.DeleteResult)
}

func NewErrorResult(err error) Result {
	return Result{
		err: err,
	}
}

func NewSuccessResult(result any) Result {
	return Result{
		result: result,
	}
}

func ToBsonDoc(v any) (doc bson.D, err error) {
	data, err := bson.Marshal(v)
	if err != nil {
		return nil, err
	}
	var bsonDoc bson.D
	err = bson.Unmarshal(data, &bsonDoc)
	return bsonDoc, err
}

func ToBsonDocForce(v any) bson.D {
	doc, err := ToBsonDoc(v)
	if err != nil {
		core.Logger.Panicf("bson conversion failed: %v", err)
	}
	return doc
}
