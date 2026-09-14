package common

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// func TestMain(m *testing.M) {
// 	setup()
// 	code := m.Run()
// 	shutdown()
// 	os.Exit(code)
// }

func Test_setClientOptions(t *testing.T) {
	connectionURI := "mongodb://localhost"
	appName := "common-test"
	client := setClientOptions(connectionURI, appName, true)
	assert.Equal(t, appName, *client.AppName)
	assert.Equal(t, ConnectTimeout, *client.ConnectTimeout)
	assert.Equal(t, MaxConnIdleTime, *client.MaxConnIdleTime)
	assert.Equal(t, ServerSelectionTimeout, *client.ServerSelectionTimeout)
	assert.Equal(t, connectionURI, client.GetURI())
}

func Test_setClientOptionsWithCredentials(t *testing.T) {
	connectionURI := "mongodb://localhost"
	appName := "common-test"
	authDB := "admin"
	userDB := "user"
	passDB := "password"
	credentials := options.Credential{AuthSource: authDB, Username: userDB, Password: passDB}
	client := setClientOptionsWithCredentials(connectionURI, appName, credentials, true)
	assert.Equal(t, appName, *client.AppName)
	assert.Equal(t, ConnectTimeout, *client.ConnectTimeout)
	assert.Equal(t, MaxConnIdleTime, *client.MaxConnIdleTime)
	assert.Equal(t, ServerSelectionTimeout, *client.ServerSelectionTimeout)
	assert.Equal(t, connectionURI, client.GetURI())
	assert.Equal(t, authDB, client.Auth.AuthSource)
	assert.Equal(t, userDB, client.Auth.Username)
	assert.Equal(t, passDB, client.Auth.Password)
}

func Test_returnClient(t *testing.T) {
	connectionURI := "localhost"
	appName := "common-test"
	client, err := ReturnClient(connectionURI, appName, true)
	assert.Nil(t, err, "should be nil")
	assert.NotNil(t, client, "should not be nil")
	err = client.Ping(context.TODO(), nil)
	assert.Nil(t, err, "should be nil")
}

func Test_Total(t *testing.T) {
	client := getClient("localhost", "common-test")
	total, err := Total(client, "test", "test", bson.M{})
	assert.Nil(t, err, "should be nil")
	assert.Equal(t, int64(0), total)
}

func Test_InsertOne(t *testing.T) {
	client := getClient("localhost", "common-test")
	insertResult, err := InsertOne(client, "test", "test", bson.E{Key: "timestamp", Value: time.Now()})
	assert.Nil(t, err, "should be nil")
	assert.NotNil(t, insertResult, "should not be nil")

}

func Test_CreateIndex(t *testing.T) {
	client := getClient("localhost", "common-test")
	dbName := "test"
	collectionName := "test_create_index"
	field := "email"
	collection := client.Database(dbName).Collection(collectionName)
	defer collection.Drop(context.TODO())

	err := CreateIndex(client, dbName, collectionName, field, true)
	assert.Nil(t, err, "should be nil")

	cursor, err := collection.Indexes().List(context.TODO())
	assert.Nil(t, err, "should be nil")

	var indexes []bson.M
	err = cursor.All(context.TODO(), &indexes)
	assert.Nil(t, err, "should be nil")

	var found bson.M
	for _, idx := range indexes {
		if keys, ok := idx["key"].(bson.M); ok {
			if _, exists := keys[field]; exists {
				found = idx
				break
			}
		}
	}
	assert.NotNil(t, found, "index on field should exist")
	assert.Equal(t, true, found["unique"])
}

func getClient(connectionURI string, appName string) *mongo.Client {
	client, _ := ReturnClient(connectionURI, appName, true)
	return client
}
