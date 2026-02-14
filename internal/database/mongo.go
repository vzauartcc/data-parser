package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoDatabase interface {
	// Notice the change in the second parameter to match the "Lister" requirement
	Collection(name string, opts ...options.Lister[options.CollectionOptions]) MongoCollection
}

type MongoCollection interface {
	BulkWrite(ctx context.Context, models []mongo.WriteModel, opts ...options.Lister[options.BulkWriteOptions]) (*mongo.BulkWriteResult, error)
	DeleteMany(ctx context.Context, filter any, opts ...options.Lister[options.DeleteManyOptions]) (*mongo.DeleteResult, error)
	InsertOne(ctx context.Context, document any, opts ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error)
	Find(ctx context.Context, filter any, opts ...options.Lister[options.FindOptions]) (MongoCursor, error)
	FindOne(ctx context.Context, filter any, opts ...options.Lister[options.FindOneOptions]) MongoSingleResult
	UpdateByID(ctx context.Context, id any, update any, opts ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, error)
}

type MongoSingleResult interface {
	Decode(v any) error
}

type MongoCursor interface {
	All(ctx context.Context, results any) error
	Next(ctx context.Context) bool
	Decode(val any) error
	Close(ctx context.Context) error
	Err() error
}

type MongoRepo struct {
	Db *mongo.Database
}

type MongoRepoCollection struct {
	*mongo.Collection // Embedding promotes all methods like BulkWrite, InsertOne, etc.
}

func (r *MongoRepo) Collection(name string, opts ...options.Lister[options.CollectionOptions]) MongoCollection {
	return &MongoRepoCollection{
		Collection: r.Db.Collection(name, opts...),
	}
}

func (c *MongoRepoCollection) Find(ctx context.Context, filter any, opts ...options.Lister[options.FindOptions]) (MongoCursor, error) {
	// Use the name of the embedded type
	cursor, err := c.Collection.Find(ctx, filter, opts...)
	return cursor, err
}

func (c *MongoRepoCollection) FindOne(ctx context.Context, filter any, opts ...options.Lister[options.FindOneOptions]) MongoSingleResult {
	return c.Collection.FindOne(ctx, filter, opts...)
}

func NewMongoClient(uri string) (*mongo.Client, error) {
	return mongo.Connect(
		options.
			Client().
			ApplyURI(uri).
			SetConnectTimeout(5 * time.Second))
}
