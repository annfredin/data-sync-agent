package storeman

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type (

	//Provider ...
	Provider interface {
		GetDB() (*mongo.Database, error)
		Get(ctx context.Context, collectionName string, filter bson.M, findOptions *options.FindOptions) ([]bson.M, error)
		UpdateMany(ctx context.Context, collectionName string, requestData interface{}, options interface{}) (int64, error)
		DeleteMany(ctx context.Context, collectionName string, filter interface{}) (int64, error)
		Close(ctx context.Context) error
	}

	//ConnRequest -> mongo Conn Request type
	ConnRequest struct {
		Endpoint   string
		UserName   string
		Password   string
		AuthSource string
		DBName     string
	}

	//MongoProvider ->  store client
	MongoProvider struct {
		client *mongo.Client
		db     *mongo.Database
	}
)

//NewMongoProvider ...
func NewMongoProvider(request ConnRequest) (Provider, error) {
	//create empty context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// auth credentials
	credential := options.Credential{
		Username: request.UserName,
		Password: request.Password,
	}

	if len(strings.TrimSpace(request.AuthSource)) > 0 {
		credential.AuthSource = request.AuthSource
	}

	// Set client options
	clientOptions := options.Client().ApplyURI(fmt.Sprintf("mongodb://%s", request.Endpoint)).SetAuth(credential)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Check the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	// returning mongo client...
	return &MongoProvider{
		client: client,
		db:     client.Database(request.DBName),
	}, nil
}

func (mdbp *MongoProvider) GetDB() (*mongo.Database, error) {
	//returning the response back...
	return mdbp.db, nil
}

//Get -> get the collection
func (mdbp *MongoProvider) Get(ctx context.Context, collectionName string, filter bson.M, findOptions *options.FindOptions) ([]bson.M, error) {
	var result []bson.M

	collection := mdbp.db.Collection(collectionName)

	cur, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}

	//this will automatically close the cur, so no need to explicitly close cursor..
	//[ defer cur.Close(ctx)]....
	err = cur.All(ctx, &result)
	if err != nil {
		return nil, err
	}

	//returning the response back...
	return result, nil
}

//InsertMany ...
func (mdbp *MongoProvider) InsertMany(ctx context.Context, collectionName string, requestData []interface{}) ([]string, error) {

	resp, err := mdbp.db.Collection(collectionName).InsertMany(ctx, requestData)
	if err != nil {
		return nil, err
	}

	responseIDs := make([]string, 0)
	for _, idObj := range resp.InsertedIDs {
		responseIDs = append(responseIDs, idObj.(primitive.ObjectID).Hex())
	}

	return responseIDs, nil
}

//InsertMany ...
func (mdbp *MongoProvider) UpdateMany(ctx context.Context, collectionName string, requestData interface{}, options interface{}) (int64, error) {

	resp, err := mdbp.db.Collection(collectionName).UpdateMany(ctx, requestData, options)
	if err != nil {
		return 0, err
	}

	return resp.ModifiedCount, nil
}

//DeleteMany ...
func (mdbp *MongoProvider) DeleteMany(ctx context.Context, collectionName string, filter interface{}) (int64, error) {
	resp, err := mdbp.db.Collection(collectionName).DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}

	return resp.DeletedCount, nil
}

//Close -> used to close the connection from mongo
func (mdbp *MongoProvider) Close(ctx context.Context) error {
	return mdbp.client.Disconnect(ctx)
}
