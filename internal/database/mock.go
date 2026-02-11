//go:build !codeanalysis

package database

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MockCollection struct {
	BulkModels    []mongo.WriteModel
	DeletedFilter any
	MockData      []any
}

func (m *MockCollection) BulkWrite(
	_ context.Context,
	models []mongo.WriteModel,
	_ ...options.Lister[options.BulkWriteOptions],
) (*mongo.BulkWriteResult, error) {
	m.BulkModels = append(m.BulkModels, models...)

	return &mongo.BulkWriteResult{}, nil
}

func (m *MockCollection) DeleteMany(
	_ context.Context,
	filter any,
	_ ...options.Lister[options.DeleteManyOptions],
) (*mongo.DeleteResult, error) {
	m.DeletedFilter = filter

	return &mongo.DeleteResult{}, nil
}

func (m *MockCollection) InsertOne(_ context.Context, _ any, _ ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error) {
	return &mongo.InsertOneResult{InsertedID: "mock-id"}, nil
}

type MockMongoDatabase struct {
	Collections map[string]*MockCollection
}

func (m *MockMongoDatabase) Collection(name string, _ ...options.Lister[options.CollectionOptions]) MongoCollection {
	return m.Collections[name]
}

type MockCursor struct {
	Data []any
	err  error
}

func (m *MockCursor) All(_ context.Context, results any) error {
	data, err := bson.Marshal(m.Data)
	if err != nil {
		return err
	}

	return bson.Unmarshal(data, results)
}

func (m *MockCursor) Next(_ context.Context) bool { return false }

func (m *MockCursor) Decode(_ any) error { return nil }

func (m *MockCursor) Close(_ context.Context) error { return nil }

func (m *MockCursor) Err() error { return m.err }

func (m *MockCollection) Find(_ context.Context, _ any, _ ...options.Lister[options.FindOptions]) (MongoCursor, error) {
	return &MockCursor{Data: m.MockData}, nil
}

type MockSingleResult struct {
	Data any
	Err  error
}

func (m *MockSingleResult) Decode(val any) error {
	if m.Err != nil {
		return m.Err
	}

	bytes, _ := bson.Marshal(m.Data)

	return bson.Unmarshal(bytes, val)
}

func (m *MockCollection) FindOne(_ context.Context, _ any, _ ...options.Lister[options.FindOneOptions]) MongoSingleResult {
	return &MockSingleResult{Data: bson.M{"cid": 1234567}}
}

func (m *MockCollection) UpdateByID(_ context.Context, _ any, _ any, _ ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, error) {
	return &mongo.UpdateResult{MatchedCount: 1, ModifiedCount: 1}, nil
}
