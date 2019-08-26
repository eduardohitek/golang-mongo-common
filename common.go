package common

import (
	"context"
	"log"
	"reflect"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func RetornarCliente(url string) *mongo.Client {
	clientOptions := options.Client().ApplyURI("mongodb://" + url)
	client, err := mongo.NewClient(clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Connect(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	return client
}

func RetornarClienteSeguro(url string, authDB string, user string, password string, appName string) *mongo.Client {
	credentials := options.Credential{AuthSource: authDB, Username: user, Password: password}
	connectionOptions := options.Client().ApplyURI("mongodb://" + url).SetAppName(appName).SetAuth(credentials).SetConnectTimeout(5 * time.Second)
	client, err := mongo.NewClient(connectionOptions)
	if err != nil {
		log.Fatal("Erro ao efetuar conexão com o DB", err.Error())
		return nil
	}
	err = client.Connect(context.Background())
	if err != nil {
		log.Fatal("Erro ao efetuar conexão com o DB", err.Error())
	}
	return client
}

func Total(nomeDB string, nomeColecao string, client *mongo.Client, filtro interface{}) int64 {
	collection := client.Database(nomeDB).Collection(nomeColecao)
	total, err := collection.CountDocuments(context.TODO(), filtro)
	log.Println(total, err)
	return total
}

func Deletar(nomeDB string, nomeColecao string, client *mongo.Client, insertedID interface{}) {
	collection := client.Database(nomeDB).Collection(nomeColecao)
	filtro := bson.M{"_id": insertedID}
	a, b := collection.DeleteOne(context.TODO(), filtro)
	log.Println(a, b)
}

func AtualizarPeloId(nomeDB string, nomeColecao string, client *mongo.Client, insertedID interface{}, campoAtualizado interface{}) {
	collection := client.Database(nomeDB).Collection(nomeColecao)
	atualizacao := bson.D{{Key: "$set", Value: campoAtualizado}}
	filtro := bson.M{"_id": insertedID}
	log.Println(filtro)
	a, b := collection.UpdateOne(context.TODO(), filtro, atualizacao)
	log.Println(a, b)
}

func Atualizar(nomeDB string, nomeColecao string, client *mongo.Client, filtro interface{}, campoAtualizado interface{}) {
	collection := client.Database(nomeDB).Collection(nomeColecao)
	atualizacao := bson.D{{Key: "$set", Value: campoAtualizado}}
	log.Println(filtro)
	a, b := collection.UpdateOne(context.TODO(), filtro, atualizacao)
	log.Println(a, b)
}

func Adicionar(ctx context.Context, nomeDB string, nomeColecao string, documento interface{}, client *mongo.Client) interface{} {
	collection := client.Database(nomeDB).Collection(nomeColecao)
	c := context.TODO()
	a, b := collection.InsertOne(c, documento)
	log.Println(a, b)
	return a.InsertedID
}

func RetornarUm(nomeDB string, nomeColecao string, modelo interface{}, client *mongo.Client, filtro bson.M) {
	collection := client.Database(nomeDB).Collection(nomeColecao)
	a := collection.FindOne(context.TODO(), filtro)
	a.Decode(modelo)
}

func RetornarTodos(ctx context.Context, nomeDB string,
	nomeColecao string, modelo interface{}, client *mongo.Client, filtro bson.M) interface{} {

	collection := client.Database(nomeDB).Collection(nomeColecao)
	cur, err := collection.Find(context.TODO(), filtro)
	if err != nil {
		log.Println("Find cur", err)
	}
	rv := reflect.ValueOf(modelo).Elem()
	sv := rv.Slice(0, rv.Cap())

	for cur.Next(context.Background()) {
		pev := reflect.New(sv.Type().Elem())
		if err := cur.Decode(pev.Interface()); err != nil {
			return err
		}

		sv = reflect.Append(sv, pev.Elem())
	}

	rv.Set(sv)
	return cur.Err()
}
