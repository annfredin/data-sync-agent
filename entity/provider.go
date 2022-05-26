package entity

import (
	"context"
	"fmt"
	"time"

	"data-sync-agent/utils/logger"

	"go.mongodb.org/mongo-driver/mongo/options"

	"go.mongodb.org/mongo-driver/mongo"

	sm "data-sync-agent/entity/storeman"
	"data-sync-agent/helper"
	model "data-sync-agent/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

var storeProvider sm.Provider

func init() {
	InitializeStoreDataProvider()
}

//InitializeStoreDataProvider ...a
func InitializeStoreDataProvider() sm.Provider {

	endPoint := helper.GetEnv(helper.MongoEndPoint)
	userName := helper.GetEnv(helper.MongoUserName)
	password := helper.GetEnv(helper.MongoPassword)
	dbName := helper.GetEnv(helper.MongoDBName)
	authDB := helper.GetEnv(helper.MongoAuthDB)


	if nOK := helper.HasEmpty(endPoint,userName,password,dbName,authDB); nOK {
		logger.Log().Fatal("Mongo Server Connection Secret Error")
		return nil
	}


	mdProvider, err := sm.NewMongoProvider(sm.ConnRequest{
		Endpoint:   endPoint,
		UserName:   userName,
		Password:   password,
		DBName:     dbName,
		AuthSource: authDB,
	})

	if err != nil {
		logger.Log().With(zap.Error(err)).Error(fmt.Sprintf("Mongo Server Connection Error EndPoint %s", endPoint))

		return nil
	}

	logger.Log().Info(fmt.Sprintf("Mongo Server started with EndPoint %s", endPoint))

	//assigning to global variable(must)
	storeProvider = mdProvider

	return mdProvider
}

//public operation fns...

//Get ...
func Get(ctx context.Context, collectionName string, filter bson.M, findOptions *options.FindOptions) ([]bson.M, error) {
	return storeProvider.Get(ctx, collectionName, filter, findOptions)
}

//BulkWrite ...
func BulkWrite(collectionName string, requestActiveData []model.MongoDeviceData, requestInActiveData []model.MongoDeviceData) (int64, error) {

	db, _ := storeProvider.GetDB()
	collection := db.Collection(collectionName)

	defaultColumnsToInsert := bson.M{"createdon": time.Now().UTC(), "insertdatetime": time.Now().UTC(), "ignitionstatus": 0, "speed": 0, "latitude": 0, "longitude": 0, "recordstatus": 0, "devicedatetime": time.Now().UTC(), "mongoid": primitive.NewObjectIDFromTimestamp(time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC))}

	var operations []mongo.WriteModel
	for _, d := range requestActiveData {
		operation := mongo.NewUpdateOneModel()
		operation.SetFilter(bson.M{"_id": d.ID})
		operation.SetUpdate(bson.M{"$set": bson.M{
			"providertenantuids": d.ProviderTenantUIDs, "tenantuid": d.TenantUID,
			"tenantgroupuid": d.TenantGroupUID, "devicetypeid": d.DeviceTypeID,
			"communicationgroupid": d.CommunicationGroupID,
			"tenantname":           d.TenantName, "vehicleid": d.VehicleID,
			"diversiondetails": d.DiversionDetails},
			"$setOnInsert": defaultColumnsToInsert,
		})

		// Set Upsert flag option to turn the update operation to upsert
		operation.SetUpsert(true)
		operations = append(operations, operation)
	}

	for _, d := range requestInActiveData {
		operation := mongo.NewDeleteOneModel()
		operation.SetFilter(bson.M{"_id": d.ID})
		operations = append(operations, operation)
	}

	// Specify an option to turn the bulk insertion in order of operation
	bulkOption := options.BulkWriteOptions{}
	bulkOption.SetOrdered(true)
	result, err := collection.BulkWrite(context.TODO(), operations, &bulkOption)
	if err != nil {
		return 0, err
	}

	return (result.InsertedCount + result.UpsertedCount + result.DeletedCount), nil
}

//BulkWriteUpdate ...
func BulkWriteUpdate(collectionName string, requestActiveData []model.MongoDeviceData) (int64, error) {

	db, _ := storeProvider.GetDB()
	collection := db.Collection(collectionName)

	var operations []mongo.WriteModel
	for _, d := range requestActiveData {
		operation := mongo.NewUpdateOneModel()
		operation.SetFilter(bson.M{"_id": d.ID})
		operation.SetUpdate(bson.M{"$set": bson.M{"vehicleid": d.VehicleID}})

		operations = append(operations, operation)
	}

	// Specify an option to turn the bulk insertion in order of operation
	bulkOption := options.BulkWriteOptions{}
	bulkOption.SetOrdered(true)
	result, err := collection.BulkWrite(context.TODO(), operations, &bulkOption)
	if err != nil {
		return 0, err
	}

	return (result.InsertedCount + result.UpsertedCount), nil
}

//BulkWrite ...
func BulkWriteUnAuth(collectionName string, requestActiveData []*model.UnAuthDeviceResponse) (int64, error) {

	db, _ := storeProvider.GetDB()
	collection := db.Collection(collectionName)

	var operations []mongo.WriteModel
	for _, d := range requestActiveData {
		operation := mongo.NewUpdateOneModel()
		operation.SetFilter(bson.M{"_id": d.DeviceID})
		operation.SetUpdate(bson.M{"$set": bson.M{
			"applicationserverid": d.ApplicationServerID, "active": d.Active,
			"devicecount": d.DeviceCount},
		})

		// Set Upsert flag option to turn the update operation to upsert
		operation.SetUpsert(false)
		operations = append(operations, operation)
	}

	// Specify an option to turn the bulk insertion in order of operation
	bulkOption := options.BulkWriteOptions{}
	bulkOption.SetOrdered(true)
	result, err := collection.BulkWrite(context.TODO(), operations, &bulkOption)
	if err != nil {
		return 0, err
	}

	return result.ModifiedCount, nil
}

//DeleteMany ...
func UpdateMany(ctx context.Context, collectionName string, filter bson.M, options interface{}) (int64, error) {
	return storeProvider.UpdateMany(ctx, collectionName, filter, options)
}

//DeleteMany ...
func DeleteMany(ctx context.Context, collectionName string, filter bson.M) (int64, error) {
	return storeProvider.DeleteMany(ctx, collectionName, filter)
}

func Close() error {
	return storeProvider.Close(context.Background())
}
