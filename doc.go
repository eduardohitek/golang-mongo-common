// Package common provides helper functions for using MongoDB from Go, built on
// the official driver go.mongodb.org/mongo-driver/v2.
//
// The package has three groups of functions:
//
//   - Client constructors: [ReturnClient], [ReturnAuthenticatedClient] and
//     [ReturnAuthenticatedClientMongoAtlas]. They apply the default timeouts
//     ([ConnectTimeout], [ServerSelectionTimeout], [MaxConnIdleTime]), connect,
//     and ping the server before returning. Behavior is tuned with [ClientOption]
//     values such as [WithLogger] and [WithTimeouts].
//   - CRUD helpers that take (ctx, client, dbName, collectionName, ...) and forward
//     the driver's own options: [Total], [InsertOne], [InsertMany], [FindOne],
//     [FindAll], [UpdateByID], [UpdateOneByFilter], [UpdateManyByFilter],
//     [DeleteOneByID], [DeleteOneByFilter], [DeleteManyByFilter].
//   - Index management: [CreateIndex].
//
// # Migrating from v1
//
// v2 is a breaking release. Every exported function documents its v1 and v2
// signatures in a "Migrating from v1:" block. The mechanical changes are:
//
//  1. Go 1.25 or newer is required.
//  2. Import path: github.com/eduardohitek/golang-mongo-common becomes
//     github.com/eduardohitek/golang-mongo-common/v2. The package name is still common.
//  3. Driver: the caller's own imports of go.mongodb.org/mongo-driver/... must move
//     to go.mongodb.org/mongo-driver/v2/...; primitive.ObjectID becomes bson.ObjectID,
//     the primitive package is gone, and options are builders
//     (options.FindOne().SetProjection(...)). When decoding into bson.M or any,
//     nested documents now come back as bson.D instead of bson.M.
//  4. Context: every function takes ctx context.Context as its first argument,
//     replacing the internal context.TODO().
//  5. Errors: constructors return errors instead of calling log.Fatal, and they ping
//     the server, so an unreachable database is reported at construction time.
//  6. Logging: the trailing createMonitor bool is removed. Use
//     WithLogger(logger) to log commands at Debug level; omit it for no logging.
//  7. Reads: FindOne and FindAll are generic. Replace the model argument and the
//     type assertion with a type parameter: FindOne[User](...) returns (User, error)
//     and FindAll[User](...) returns ([]User, error).
//  8. Not found: check errors.Is(err, mongo.ErrNoDocuments) after FindOne.
//  9. Filters and ids are typed any (bson.M still works; bson.D keeps key order),
//     and each helper accepts the driver's options variadically.
package common
