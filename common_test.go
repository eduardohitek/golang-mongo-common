package common

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const testDB = "test"

type person struct {
	ID   bson.ObjectID `bson:"_id,omitempty"`
	Name string        `bson:"name"`
	Age  int           `bson:"age"`
}

func Test_setClientOptions(t *testing.T) {
	connectionURI := "mongodb://localhost"
	appName := "common-test"
	client := setClientOptions(connectionURI, appName, newClientConfig(nil))
	assert.Equal(t, appName, *client.AppName)
	assert.Equal(t, ConnectTimeout, *client.ConnectTimeout)
	assert.Equal(t, MaxConnIdleTime, *client.MaxConnIdleTime)
	assert.Equal(t, ServerSelectionTimeout, *client.ServerSelectionTimeout)
	assert.Equal(t, connectionURI, client.GetURI())
	assert.Nil(t, client.Monitor, "monitor should be off without WithLogger")
}

func Test_setClientOptions_withOptions(t *testing.T) {
	cfg := newClientConfig([]ClientOption{
		WithLogger(slog.Default()),
		WithTimeouts(time.Second, 2*time.Second, 3*time.Second),
	})
	client := setClientOptions("mongodb://localhost", "common-test", cfg)
	assert.Equal(t, time.Second, *client.ConnectTimeout)
	assert.Equal(t, 2*time.Second, *client.ServerSelectionTimeout)
	assert.Equal(t, 3*time.Second, *client.MaxConnIdleTime)
	assert.NotNil(t, client.Monitor)
}

func Test_setClientOptionsWithCredentials(t *testing.T) {
	connectionURI := "mongodb://localhost"
	appName := "common-test"
	authDB := "admin"
	userDB := "user"
	passDB := "password"
	credentials := options.Credential{AuthSource: authDB, Username: userDB, Password: passDB}
	client := setClientOptionsWithCredentials(connectionURI, appName, credentials, newClientConfig(nil))
	assert.Equal(t, appName, *client.AppName)
	assert.Equal(t, ConnectTimeout, *client.ConnectTimeout)
	assert.Equal(t, MaxConnIdleTime, *client.MaxConnIdleTime)
	assert.Equal(t, ServerSelectionTimeout, *client.ServerSelectionTimeout)
	assert.Equal(t, connectionURI, client.GetURI())
	assert.Equal(t, authDB, client.Auth.AuthSource)
	assert.Equal(t, userDB, client.Auth.Username)
	assert.Equal(t, passDB, client.Auth.Password)
}

func Test_atlasURI(t *testing.T) {
	uri := atlasURI("cluster0.example.mongodb.net", "us@er", "p@ss:w/rd", "mydb")
	assert.Equal(t,
		"mongodb+srv://us%40er:p%40ss%3Aw%2Frd@cluster0.example.mongodb.net/mydb?retryWrites=true&w=majority",
		uri)

	// The escaped URI must decode back to the raw credentials. (options.ApplyURI is not
	// used here because mongodb+srv makes it resolve the fake host via DNS.)
	parsed, err := url.Parse(uri)
	require.NoError(t, err)
	assert.Equal(t, "us@er", parsed.User.Username())
	password, _ := parsed.User.Password()
	assert.Equal(t, "p@ss:w/rd", password)
	assert.Equal(t, "cluster0.example.mongodb.net", parsed.Host)
}

func Test_returnClient(t *testing.T) {
	client, err := ReturnClient(context.Background(), "localhost", "common-test")
	require.NoError(t, err)
	require.NotNil(t, client)
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	assert.NoError(t, client.Ping(context.Background(), nil))
}

func Test_returnClient_unreachable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	client, err := ReturnClient(ctx, "127.0.0.1:1", "common-test",
		WithTimeouts(200*time.Millisecond, 200*time.Millisecond, time.Second))
	assert.Error(t, err)
	assert.Nil(t, client)
}

func Test_WithLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client, err := ReturnClient(context.Background(), "localhost", "common-test", WithLogger(logger))
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })

	_, err = Total(context.Background(), client, testDB, newCollection(t, client), bson.M{})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "command=aggregate")
	assert.False(t, strings.Contains(buf.String(), "command=endSessions"))
}

func Test_Total(t *testing.T) {
	client := getClient(t)
	coll := newCollection(t, client)
	total, err := Total(context.Background(), client, testDB, coll, bson.M{})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)

	_, err = InsertOne(context.Background(), client, testDB, coll, person{Name: "a"})
	require.NoError(t, err)
	total, err = Total(context.Background(), client, testDB, coll, bson.D{{Key: "name", Value: "a"}})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
}

func Test_InsertOne(t *testing.T) {
	client := getClient(t)
	insertResult, err := InsertOne(context.Background(), client, testDB, newCollection(t, client),
		bson.D{{Key: "timestamp", Value: time.Now()}})
	assert.NoError(t, err)
	require.NotNil(t, insertResult)
	assert.IsType(t, bson.ObjectID{}, insertResult.InsertedID)
}

func Test_InsertMany(t *testing.T) {
	client := getClient(t)
	coll := newCollection(t, client)
	result, err := InsertMany(context.Background(), client, testDB, coll,
		[]any{person{Name: "a"}, person{Name: "b"}})
	require.NoError(t, err)
	assert.Len(t, result.InsertedIDs, 2)
}

func Test_FindOne(t *testing.T) {
	client := getClient(t)
	coll := newCollection(t, client)
	_, err := InsertOne(context.Background(), client, testDB, coll, person{Name: "ana", Age: 30})
	require.NoError(t, err)

	found, err := FindOne[person](context.Background(), client, testDB, coll, bson.M{"name": "ana"})
	require.NoError(t, err)
	assert.Equal(t, 30, found.Age)

	projected, err := FindOne[person](context.Background(), client, testDB, coll, bson.M{"name": "ana"},
		options.FindOne().SetProjection(bson.M{"age": 0}))
	require.NoError(t, err)
	assert.Equal(t, 0, projected.Age)

	_, err = FindOne[person](context.Background(), client, testDB, coll, bson.M{"name": "nobody"})
	assert.True(t, errors.Is(err, mongo.ErrNoDocuments))
}

func Test_FindAll(t *testing.T) {
	client := getClient(t)
	coll := newCollection(t, client)

	empty, err := FindAll[person](context.Background(), client, testDB, coll, bson.M{})
	require.NoError(t, err)
	assert.NotNil(t, empty)
	assert.Len(t, empty, 0)

	_, err = InsertMany(context.Background(), client, testDB, coll,
		[]any{person{Name: "a", Age: 1}, person{Name: "b", Age: 2}, person{Name: "c", Age: 3}})
	require.NoError(t, err)

	all, err := FindAll[person](context.Background(), client, testDB, coll, bson.M{"age": bson.M{"$gte": 2}},
		options.Find().SetSort(bson.D{{Key: "age", Value: 1}}))
	require.NoError(t, err)
	require.Len(t, all, 2)
	assert.Equal(t, "b", all[0].Name)
	assert.Equal(t, "c", all[1].Name)
}

func Test_Update(t *testing.T) {
	client := getClient(t)
	coll := newCollection(t, client)
	ctx := context.Background()
	res, err := InsertMany(ctx, client, testDB, coll,
		[]any{person{Name: "a", Age: 1}, person{Name: "b", Age: 1}, person{Name: "c", Age: 1}})
	require.NoError(t, err)

	updated, err := UpdateByID(ctx, client, testDB, coll, res.InsertedIDs[0], bson.M{"age": 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), updated.ModifiedCount)

	updated, err = UpdateOneByFilter(ctx, client, testDB, coll, bson.M{"name": "b"}, bson.M{"age": 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), updated.ModifiedCount)

	updated, err = UpdateManyByFilter(ctx, client, testDB, coll, bson.M{}, bson.M{"name": "z"})
	require.NoError(t, err)
	assert.Equal(t, int64(3), updated.ModifiedCount)

	first, err := FindOne[person](ctx, client, testDB, coll, bson.M{"_id": res.InsertedIDs[0]})
	require.NoError(t, err)
	assert.Equal(t, 10, first.Age)
	assert.Equal(t, "z", first.Name)
}

func Test_Delete(t *testing.T) {
	client := getClient(t)
	coll := newCollection(t, client)
	ctx := context.Background()
	res, err := InsertMany(ctx, client, testDB, coll,
		[]any{person{Name: "a"}, person{Name: "b"}, person{Name: "c"}, person{Name: "c"}})
	require.NoError(t, err)

	deleted, err := DeleteOneByID(ctx, client, testDB, coll, res.InsertedIDs[0])
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted.DeletedCount)

	deleted, err = DeleteOneByFilter(ctx, client, testDB, coll, bson.M{"name": "b"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted.DeletedCount)

	deleted, err = DeleteManyByFilter(ctx, client, testDB, coll, bson.M{"name": "c"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted.DeletedCount)

	total, err := Total(ctx, client, testDB, coll, bson.M{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
}

func Test_CreateIndex(t *testing.T) {
	client := getClient(t)
	collectionName := newCollection(t, client)
	field := "email"
	collection := client.Database(testDB).Collection(collectionName)

	err := CreateIndex(context.Background(), client, testDB, collectionName, field, true)
	require.NoError(t, err)

	cursor, err := collection.Indexes().List(context.Background())
	require.NoError(t, err)

	var indexes []bson.M
	err = cursor.All(context.Background(), &indexes)
	require.NoError(t, err)

	// With driver v2, documents nested in a bson.M decode as bson.D.
	var found bson.M
	for _, idx := range indexes {
		if keys, ok := idx["key"].(bson.D); ok {
			for _, key := range keys {
				if key.Key == field {
					found = idx
				}
			}
		}
	}
	require.NotNil(t, found, "index on field should exist")
	assert.Equal(t, true, found["unique"])
}

// getClient connects to the local MongoDB used by the tests and disconnects on cleanup.
func getClient(t *testing.T) *mongo.Client {
	t.Helper()
	client, err := ReturnClient(context.Background(), "localhost", "common-test")
	require.NoError(t, err, "tests need a MongoDB on localhost:27017")
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	return client
}

// newCollection returns a collection name unique to the test and drops it on cleanup.
func newCollection(t *testing.T, client *mongo.Client) string {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	collection := client.Database(testDB).Collection(name)
	_ = collection.Drop(context.Background())
	t.Cleanup(func() { _ = collection.Drop(context.Background()) })
	return name
}
