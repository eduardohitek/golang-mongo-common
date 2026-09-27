package common_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"

	common "github.com/eduardohitek/golang-mongo-common/v2"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type User struct {
	ID    bson.ObjectID `bson:"_id,omitempty"`
	Name  string        `bson:"name"`
	Email string        `bson:"email"`
}

// These examples need a running MongoDB, so they have no Output line:
// they are compiled by go test/go vet but not executed.

func ExampleReturnClient() {
	ctx := context.Background()

	// v1: client, _ := common.ReturnClient("localhost", "my-app", true)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client, err := common.ReturnClient(ctx, "localhost", "my-app", common.WithLogger(logger))
	if err != nil {
		log.Fatal(err) // v2 returns the error; deciding to exit is up to the caller
	}
	defer client.Disconnect(ctx)
}

func ExampleFindOne() {
	ctx := context.Background()
	client, err := common.ReturnClient(ctx, "localhost", "my-app")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	// v1: res, err := common.FindOne(client, "app", "users", &User{}, bson.M{"email": "a@b.c"}, nil)
	//     user := res.(*User)
	user, err := common.FindOne[User](ctx, client, "app", "users", bson.M{"email": "a@b.c"})
	if errors.Is(err, mongo.ErrNoDocuments) {
		fmt.Println("not found")
		return
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(user.Name)
}

func ExampleFindAll() {
	ctx := context.Background()
	client, err := common.ReturnClient(ctx, "localhost", "my-app")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	// v1: res, err := common.FindAll(client, "app", "users", []User{}, bson.M{})
	//     users := res.([]User)
	users, err := common.FindAll[User](ctx, client, "app", "users", bson.M{},
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}).SetLimit(10))
	if err != nil {
		log.Fatal(err)
	}
	for _, u := range users {
		fmt.Println(u.Name)
	}
}

func ExampleUpdateByID() {
	ctx := context.Background()
	client, err := common.ReturnClient(ctx, "localhost", "my-app")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	// v1: id is primitive.ObjectID; v2: bson.ObjectID.
	id, _ := bson.ObjectIDFromHex("64b7f0c2a1b2c3d4e5f60718")

	// v1: common.UpdateByID(client, "app", "users", id, bson.M{"name": "New"})
	result, err := common.UpdateByID(ctx, client, "app", "users", id, bson.M{"name": "New"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.ModifiedCount)
}
