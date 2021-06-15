package common

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const ConnectTimeout = 10 * time.Second
const MaxConnIdleTime = 15 * time.Second
const ServerSelectionTimeout = 10 * time.Second

// Sets the default options for the Client.
func setClientOptions(connectionURI string, appName string) *options.ClientOptions {
	clientOptions := options.Client()
	clientOptions.ApplyURI(connectionURI)
	clientOptions.SetConnectTimeout(ConnectTimeout)
	clientOptions.SetAppName(appName)
	clientOptions.SetMaxConnIdleTime(MaxConnIdleTime)
	clientOptions.SetServerSelectionTimeout(ServerSelectionTimeout)
	return clientOptions
}

// Sets the default option for the Client with credentials
func setClientOptionsWithCredentials(connectionURI string, appName string,
	credentials options.Credential) *options.ClientOptions {

	clientOptions := setClientOptions(connectionURI, appName)
	clientOptions.SetAuth(credentials)
	return clientOptions
}

// Returns a non-authenticated mongodb client.
func ReturnClient(url string, appName string) (*mongo.Client, error) {
	connectionURI := fmt.Sprintf("mongodb://%s", url)
	clientOptions := setClientOptions(connectionURI, appName)
	client, err := mongo.NewClient(clientOptions)
	if err != nil {
		log.Fatal("Error on creating the database client. ", err.Error())
		return nil, err
	}
	err = client.Connect(context.TODO())
	if err != nil {
		log.Fatal("Error on connection to the database. ", err.Error())
		return nil, err
	}
	return client, nil
}

// Returns an authenticated mongodb client.
func ReturnAuthenticatedClient(url string, authDB string, user string, password string,
	appName string) (*mongo.Client, error) {

	credentials := options.Credential{AuthSource: authDB, Username: user, Password: password}
	connectionURI := fmt.Sprintf("mongodb://%s", url)
	clientOptions := setClientOptionsWithCredentials(connectionURI, appName, credentials)
	client, err := mongo.NewClient(clientOptions)
	if err != nil {
		log.Fatal("Error on creating the database client", err.Error())
		return nil, err
	}
	err = client.Connect(context.TODO())
	if err != nil {
		log.Fatal("Error on connection to the database", err.Error())
		return nil, err
	}
	return client, nil
}

// Return an autheticated mongodb client for Mongo Atlas
func ReturnAuthenticatedClientMongoAtlas(url string, user string, password string, db string,
	appName string) (*mongo.Client, error) {

	connectionURI := fmt.Sprintf("mongodb+srv://%s:%s@%s/%s?retryWrites=true&w=majority", user, password, url, db)
	clientOptions := setClientOptions(connectionURI, appName)
	client, err := mongo.NewClient(clientOptions)
	if err != nil {
		log.Fatal("Error on creating the database client", err.Error())
		return nil, err
	}
	err = client.Connect(context.TODO())
	if err != nil {
		log.Fatal("Error on connection to the database", err.Error())
		return nil, err
	}
	return client, nil
}

// Returns the total number of documents in one collection
func Total(client *mongo.Client, dbName string, collectionName string, filter bson.M) (int64, error) {
	collection := client.Database(dbName).Collection(collectionName)
	total, err := collection.CountDocuments(context.TODO(), filter)
	return total, err
}

// Deletes a document by the informed ID
func DeleteOneByID(client *mongo.Client, dbName string, collectionName string,
	insertedID interface{}) (*mongo.DeleteResult, error) {

	collection := client.Database(dbName).Collection(collectionName)
	filter := bson.M{"_id": insertedID}
	deleteResult, err := collection.DeleteOne(context.TODO(), filter)
	return deleteResult, err
}

// Deletes one document by the informed filter
func DeleteOneByFilter(client *mongo.Client, dbName string, collectionName string,
	filter bson.M) (*mongo.DeleteResult, error) {

	collection := client.Database(dbName).Collection(collectionName)
	deleteResult, err := collection.DeleteOne(context.TODO(), filter)
	return deleteResult, err
}

// Deletes one or many document by the informed filter
func DeleteManyByFilter(client *mongo.Client, dbName string, collectionName string,
	filter bson.M) (*mongo.DeleteResult, error) {

	collection := client.Database(dbName).Collection(collectionName)
	deleteResult, err := collection.DeleteMany(context.TODO(), filter)
	return deleteResult, err
}

func UpdateByID(client *mongo.Client, dbName string, collectionName string, insertedID interface{},
	campoAtualizado interface{}) (*mongo.UpdateResult, error) {

	collection := client.Database(dbName).Collection(collectionName)
	atualizacao := bson.D{{Key: "$set", Value: campoAtualizado}}
	filter := bson.M{"_id": insertedID}
	updateResult, err := collection.UpdateOne(context.TODO(), filter, atualizacao)
	return updateResult, err
}

func UpdateByFilter(client *mongo.Client, dbName string, collectionName string, filter bson.M,
	campoAtualizado interface{}) (*mongo.UpdateResult, error) {

	collection := client.Database(dbName).Collection(collectionName)
	atualizacao := bson.D{{Key: "$set", Value: campoAtualizado}}
	updateResult, err := collection.UpdateOne(context.TODO(), filter, atualizacao)
	return updateResult, err
}

func InsertOne(client *mongo.Client, dbName string, collectionName string,
	model interface{}) (*mongo.InsertOneResult, error) {

	collection := client.Database(dbName).Collection(collectionName)
	insertResult, err := collection.InsertOne(context.TODO(), model)
	return insertResult, err
}

func InsertMany(client *mongo.Client, dbName string, collectionName string,
	models []interface{}) (*mongo.InsertManyResult, error) {

	collection := client.Database(dbName).Collection(collectionName)
	insertResult, err := collection.InsertMany(context.TODO(), models)
	return insertResult, err
}

func FindOne(client *mongo.Client, dbName string, collectionName string, model interface{},
	filter bson.M, findOption *options.FindOneOptions) (interface{}, error) {

	collection := client.Database(dbName).Collection(collectionName)
	a := collection.FindOne(context.TODO(), filter, findOption)
	err := a.Decode(model)
	return model, err
}

func FindAll(client *mongo.Client, dbName string, collectionName string, model interface{},
	filter bson.M) (interface{}, error) {

	collection := client.Database(dbName).Collection(collectionName)
	cur, err := collection.Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}
	err = cur.All(context.TODO(), &model)
	if err != nil {
		return nil, err
	}

	return model, err
}
