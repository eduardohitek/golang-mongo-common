package common

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Default timeouts applied to every client created by this package.
// They can be overridden per client with [WithTimeouts].
const ConnectTimeout = 10 * time.Second
const MaxConnIdleTime = 15 * time.Second
const ServerSelectionTimeout = 10 * time.Second

// ClientOption configures the clients created by [ReturnClient],
// [ReturnAuthenticatedClient] and [ReturnAuthenticatedClientMongoAtlas].
//
// Migrating from v1:
//
//	v1 had no options; the only knob was the positional `createMonitor bool`.
//	v2 replaces it with variadic ClientOption values passed at the end of each constructor.
type ClientOption func(*clientConfig)

type clientConfig struct {
	logger                 *slog.Logger
	connectTimeout         time.Duration
	serverSelectionTimeout time.Duration
	maxConnIdleTime        time.Duration
}

func newClientConfig(opts []ClientOption) clientConfig {
	cfg := clientConfig{
		connectTimeout:         ConnectTimeout,
		serverSelectionTimeout: ServerSelectionTimeout,
		maxConnIdleTime:        MaxConnIdleTime,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// WithLogger logs every command sent to MongoDB (except endSessions) at Debug level
// on the given logger. A nil logger disables command logging, which is the default.
//
// Migrating from v1:
//
//	v1: ReturnClient(url, appName, true)   // logged commands with log.Println
//	v2: ReturnClient(ctx, url, appName, common.WithLogger(slog.Default()))
//
//	v1: ReturnClient(url, appName, false)
//	v2: ReturnClient(ctx, url, appName)
//
// Note that slog.Default() drops Debug records unless its level is lowered;
// pass a logger whose handler enables slog.LevelDebug to see the commands.
func WithLogger(logger *slog.Logger) ClientOption {
	return func(cfg *clientConfig) {
		cfg.logger = logger
	}
}

// WithTimeouts overrides the default [ConnectTimeout], [ServerSelectionTimeout]
// and [MaxConnIdleTime] for one client.
//
// Migrating from v1: new in v2; v1 always used the package constants.
func WithTimeouts(connect, serverSelection, maxConnIdle time.Duration) ClientOption {
	return func(cfg *clientConfig) {
		cfg.connectTimeout = connect
		cfg.serverSelectionTimeout = serverSelection
		cfg.maxConnIdleTime = maxConnIdle
	}
}

// Sets the default options for the Client.
func setClientOptions(connectionURI string, appName string, cfg clientConfig) *options.ClientOptions {
	clientOptions := options.Client()
	clientOptions.ApplyURI(connectionURI)
	clientOptions.SetConnectTimeout(cfg.connectTimeout)
	clientOptions.SetAppName(appName)
	clientOptions.SetMaxConnIdleTime(cfg.maxConnIdleTime)
	clientOptions.SetServerSelectionTimeout(cfg.serverSelectionTimeout)
	if cfg.logger != nil {
		logger := cfg.logger
		monitor := &event.CommandMonitor{
			Started: func(ctx context.Context, e *event.CommandStartedEvent) {
				if e.CommandName != "endSessions" {
					logger.DebugContext(ctx, "mongodb command",
						"command", e.CommandName,
						"database", e.DatabaseName,
						"body", e.Command.String())
				}
			},
		}
		clientOptions.SetMonitor(monitor)
	}
	return clientOptions
}

// Sets the default option for the Client with credentials
func setClientOptionsWithCredentials(connectionURI string, appName string,
	credentials options.Credential, cfg clientConfig) *options.ClientOptions {

	clientOptions := setClientOptions(connectionURI, appName, cfg)
	clientOptions.SetAuth(credentials)
	return clientOptions
}

// Builds the mongodb+srv URI for Mongo Atlas, escaping user and password.
func atlasURI(host string, user string, password string, db string) string {
	u := url.URL{
		Scheme:   "mongodb+srv",
		User:     url.UserPassword(user, password),
		Host:     host,
		Path:     "/" + db,
		RawQuery: "retryWrites=true&w=majority",
	}
	return u.String()
}

// Connects and pings the server, so an unreachable database is reported right away.
func connect(ctx context.Context, clientOptions *options.ClientOptions) (*mongo.Client, error) {
	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("common: connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("common: ping: %w", err)
	}
	return client, nil
}

// ReturnClient returns a non-authenticated client connected to mongodb://host.
// The server is pinged using ctx before returning; on failure the error is returned
// and no client is left open.
//
// Migrating from v1:
//
//	v1: ReturnClient(url, appName string, createMonitor bool) (*mongo.Client, error)
//	v2: ReturnClient(ctx, host, appName string, opts ...ClientOption) (*mongo.Client, error)
//
//	before: client, _ := common.ReturnClient("localhost", "app", false)
//	after:  client, err := common.ReturnClient(ctx, "localhost", "app")
//
// v1 called log.Fatal on connection errors (the process exited); v2 returns the error,
// so callers must handle it. Replace createMonitor=true with [WithLogger].
func ReturnClient(ctx context.Context, host string, appName string, opts ...ClientOption) (*mongo.Client, error) {
	connectionURI := fmt.Sprintf("mongodb://%s", host)
	clientOptions := setClientOptions(connectionURI, appName, newClientConfig(opts))
	return connect(ctx, clientOptions)
}

// ReturnAuthenticatedClient returns a client connected to mongodb://host that
// authenticates as user against authDB. The server is pinged using ctx before returning.
//
// Migrating from v1:
//
//	v1: ReturnAuthenticatedClient(url, authDB, user, password, appName string, createMonitor bool) (*mongo.Client, error)
//	v2: ReturnAuthenticatedClient(ctx, host, authDB, user, password, appName string, opts ...ClientOption) (*mongo.Client, error)
//
//	before: client, _ := common.ReturnAuthenticatedClient("localhost", "admin", "u", "p", "app", false)
//	after:  client, err := common.ReturnAuthenticatedClient(ctx, "localhost", "admin", "u", "p", "app")
//
// As with [ReturnClient], errors are returned instead of calling log.Fatal.
func ReturnAuthenticatedClient(ctx context.Context, host string, authDB string, user string, password string,
	appName string, opts ...ClientOption) (*mongo.Client, error) {

	credentials := options.Credential{AuthSource: authDB, Username: user, Password: password}
	connectionURI := fmt.Sprintf("mongodb://%s", host)
	clientOptions := setClientOptionsWithCredentials(connectionURI, appName, credentials, newClientConfig(opts))
	return connect(ctx, clientOptions)
}

// ReturnAuthenticatedClientMongoAtlas returns a client connected to a Mongo Atlas
// cluster (mongodb+srv://). User and password are escaped, so they may contain
// characters such as '@', ':' or '/'. The server is pinged using ctx before returning.
//
// Migrating from v1:
//
//	v1: ReturnAuthenticatedClientMongoAtlas(url, user, password, db, appName string, createMonitor bool) (*mongo.Client, error)
//	v2: ReturnAuthenticatedClientMongoAtlas(ctx, host, user, password, db, appName string, opts ...ClientOption) (*mongo.Client, error)
//
//	before: client, _ := common.ReturnAuthenticatedClientMongoAtlas("cluster0.x.mongodb.net", "u", "p", "db", "app", false)
//	after:  client, err := common.ReturnAuthenticatedClientMongoAtlas(ctx, "cluster0.x.mongodb.net", "u", "p", "db", "app")
//
// If a v1 caller pre-escaped the password itself, remove that escaping: v2 escapes it.
func ReturnAuthenticatedClientMongoAtlas(ctx context.Context, host string, user string, password string, db string,
	appName string, opts ...ClientOption) (*mongo.Client, error) {

	clientOptions := setClientOptions(atlasURI(host, user, password, db), appName, newClientConfig(opts))
	return connect(ctx, clientOptions)
}

// Total returns the number of documents in the collection that match filter.
//
// Migrating from v1:
//
//	v1: Total(client, dbName, collectionName string, filter bson.M) (int64, error)
//	v2: Total(ctx, client, dbName, collectionName string, filter any, opts ...options.Lister[options.CountOptions]) (int64, error)
//
// ctx added as first parameter; filter accepts any (bson.M still works); otherwise unchanged.
func Total(ctx context.Context, client *mongo.Client, dbName string, collectionName string, filter any,
	opts ...options.Lister[options.CountOptions]) (int64, error) {

	collection := client.Database(dbName).Collection(collectionName)
	return collection.CountDocuments(ctx, filter, opts...)
}

// DeleteOneByID deletes the document whose _id equals id.
//
// Migrating from v1:
//
//	v1: DeleteOneByID(client, dbName, collectionName string, insertedID interface{}) (*mongo.DeleteResult, error)
//	v2: DeleteOneByID(ctx, client, dbName, collectionName string, id any, opts ...options.Lister[options.DeleteOneOptions]) (*mongo.DeleteResult, error)
//
// ctx added as first parameter; otherwise unchanged. If the id is an ObjectID,
// it is now bson.ObjectID (the driver v2 removed the primitive package).
func DeleteOneByID(ctx context.Context, client *mongo.Client, dbName string, collectionName string,
	id any, opts ...options.Lister[options.DeleteOneOptions]) (*mongo.DeleteResult, error) {

	collection := client.Database(dbName).Collection(collectionName)
	filter := bson.M{"_id": id}
	return collection.DeleteOne(ctx, filter, opts...)
}

// DeleteOneByFilter deletes the first document that matches filter.
//
// Migrating from v1:
//
//	v1: DeleteOneByFilter(client, dbName, collectionName string, filter bson.M) (*mongo.DeleteResult, error)
//	v2: DeleteOneByFilter(ctx, client, dbName, collectionName string, filter any, opts ...options.Lister[options.DeleteOneOptions]) (*mongo.DeleteResult, error)
//
// ctx added as first parameter; filter accepts any (bson.M still works); otherwise unchanged.
func DeleteOneByFilter(ctx context.Context, client *mongo.Client, dbName string, collectionName string,
	filter any, opts ...options.Lister[options.DeleteOneOptions]) (*mongo.DeleteResult, error) {

	collection := client.Database(dbName).Collection(collectionName)
	return collection.DeleteOne(ctx, filter, opts...)
}

// DeleteManyByFilter deletes every document that matches filter.
//
// Migrating from v1:
//
//	v1: DeleteManyByFilter(client, dbName, collectionName string, filter bson.M) (*mongo.DeleteResult, error)
//	v2: DeleteManyByFilter(ctx, client, dbName, collectionName string, filter any, opts ...options.Lister[options.DeleteManyOptions]) (*mongo.DeleteResult, error)
//
// ctx added as first parameter; filter accepts any (bson.M still works); otherwise unchanged.
func DeleteManyByFilter(ctx context.Context, client *mongo.Client, dbName string, collectionName string,
	filter any, opts ...options.Lister[options.DeleteManyOptions]) (*mongo.DeleteResult, error) {

	collection := client.Database(dbName).Collection(collectionName)
	return collection.DeleteMany(ctx, filter, opts...)
}

// UpdateByID applies {"$set": update} to the document whose _id equals id.
//
// Migrating from v1:
//
//	v1: UpdateByID(client, dbName, collectionName string, insertedID interface{}, updatedField interface{}) (*mongo.UpdateResult, error)
//	v2: UpdateByID(ctx, client, dbName, collectionName string, id any, update any, opts ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, error)
//
// ctx added as first parameter; the value is still wrapped in $set; otherwise unchanged.
func UpdateByID(ctx context.Context, client *mongo.Client, dbName string, collectionName string, id any,
	update any, opts ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, error) {

	collection := client.Database(dbName).Collection(collectionName)
	filter := bson.M{"_id": id}
	return collection.UpdateOne(ctx, filter, bson.D{{Key: "$set", Value: update}}, opts...)
}

// UpdateOneByFilter applies {"$set": update} to the first document that matches filter.
//
// Migrating from v1:
//
//	v1: UpdateOneByFilter(client, dbName, collectionName string, filter bson.M, updatedField interface{}) (*mongo.UpdateResult, error)
//	v2: UpdateOneByFilter(ctx, client, dbName, collectionName string, filter any, update any, opts ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, error)
//
// ctx added as first parameter; the value is still wrapped in $set; otherwise unchanged.
func UpdateOneByFilter(ctx context.Context, client *mongo.Client, dbName string, collectionName string, filter any,
	update any, opts ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, error) {

	collection := client.Database(dbName).Collection(collectionName)
	return collection.UpdateOne(ctx, filter, bson.D{{Key: "$set", Value: update}}, opts...)
}

// UpdateManyByFilter applies {"$set": update} to every document that matches filter.
//
// Migrating from v1:
//
//	v1: UpdateManyByFilter(client, dbName, collectionName string, filter bson.M, updatedField interface{}) (*mongo.UpdateResult, error)
//	v2: UpdateManyByFilter(ctx, client, dbName, collectionName string, filter any, update any, opts ...options.Lister[options.UpdateManyOptions]) (*mongo.UpdateResult, error)
//
// ctx added as first parameter; the value is still wrapped in $set; otherwise unchanged.
func UpdateManyByFilter(ctx context.Context, client *mongo.Client, dbName string, collectionName string, filter any,
	update any, opts ...options.Lister[options.UpdateManyOptions]) (*mongo.UpdateResult, error) {

	collection := client.Database(dbName).Collection(collectionName)
	return collection.UpdateMany(ctx, filter, bson.D{{Key: "$set", Value: update}}, opts...)
}

// InsertOne inserts document into the collection.
//
// Migrating from v1:
//
//	v1: InsertOne(client, dbName, collectionName string, model interface{}) (*mongo.InsertOneResult, error)
//	v2: InsertOne(ctx, client, dbName, collectionName string, document any, opts ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error)
//
// ctx added as first parameter; otherwise unchanged. InsertedID holds a
// bson.ObjectID (not primitive.ObjectID) when the driver generates the _id.
func InsertOne(ctx context.Context, client *mongo.Client, dbName string, collectionName string,
	document any, opts ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error) {

	collection := client.Database(dbName).Collection(collectionName)
	return collection.InsertOne(ctx, document, opts...)
}

// InsertMany inserts documents into the collection.
//
// Migrating from v1:
//
//	v1: InsertMany(client, dbName, collectionName string, models []interface{}) (*mongo.InsertManyResult, error)
//	v2: InsertMany(ctx, client, dbName, collectionName string, documents []any, opts ...options.Lister[options.InsertManyOptions]) (*mongo.InsertManyResult, error)
//
// ctx added as first parameter; otherwise unchanged.
func InsertMany(ctx context.Context, client *mongo.Client, dbName string, collectionName string,
	documents []any, opts ...options.Lister[options.InsertManyOptions]) (*mongo.InsertManyResult, error) {

	collection := client.Database(dbName).Collection(collectionName)
	return collection.InsertMany(ctx, documents, opts...)
}

// FindOne decodes the first document matching filter into a value of type T.
// When nothing matches, the error satisfies errors.Is(err, mongo.ErrNoDocuments).
//
// Migrating from v1:
//
//	v1: FindOne(client, dbName, collectionName string, model interface{}, filter bson.M, findOption *options.FindOneOptions) (interface{}, error)
//	v2: FindOne[T any](ctx, client, dbName, collectionName string, filter any, opts ...options.Lister[options.FindOneOptions]) (T, error)
//
//	before: res, err := common.FindOne(c, "db", "users", &User{}, bson.M{"name": "x"}, nil); u := res.(*User)
//	after:  u, err := common.FindOne[User](ctx, c, "db", "users", bson.M{"name": "x"})
//
// Drop the model argument and the type assertion; pass options such as
// options.FindOne().SetProjection(...) variadically instead of a trailing nil.
func FindOne[T any](ctx context.Context, client *mongo.Client, dbName string, collectionName string,
	filter any, opts ...options.Lister[options.FindOneOptions]) (T, error) {

	var result T
	collection := client.Database(dbName).Collection(collectionName)
	err := collection.FindOne(ctx, filter, opts...).Decode(&result)
	return result, err
}

// FindAll decodes every document matching filter into a []T. When nothing
// matches it returns an empty, non-nil slice.
//
// Migrating from v1:
//
//	v1: FindAll(client, dbName, collectionName string, model interface{}, filter bson.M) (interface{}, error)
//	v2: FindAll[T any](ctx, client, dbName, collectionName string, filter any, opts ...options.Lister[options.FindOptions]) ([]T, error)
//
//	before: res, err := common.FindAll(c, "db", "users", []User{}, bson.M{}); users := res.([]User)
//	after:  users, err := common.FindAll[User](ctx, c, "db", "users", bson.M{})
//
// Drop the model argument and the type assertion. Note that T is the element
// type (User), not the slice type ([]User).
func FindAll[T any](ctx context.Context, client *mongo.Client, dbName string, collectionName string,
	filter any, opts ...options.Lister[options.FindOptions]) ([]T, error) {

	collection := client.Database(dbName).Collection(collectionName)
	cur, err := collection.Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	results := []T{}
	if err := cur.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// CreateIndex creates an ascending index on field, optionally unique.
//
// Migrating from v1:
//
//	v1: CreateIndex(client, dbName, collectionName, field string, unique bool) error
//	v2: CreateIndex(ctx, client, dbName, collectionName, field string, unique bool) error
//
// ctx added as first parameter; otherwise unchanged.
func CreateIndex(ctx context.Context, client *mongo.Client, dbName string, collectionName string,
	field string, unique bool) error {

	index := mongo.IndexModel{
		Keys:    bson.D{{Key: field, Value: 1}}, // index in ascending order or -1 for descending order
		Options: options.Index().SetUnique(unique),
	}

	collection := client.Database(dbName).Collection(collectionName)
	_, err := collection.Indexes().CreateOne(ctx, index)
	return err
}
