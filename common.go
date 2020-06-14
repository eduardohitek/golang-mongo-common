package main

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

//DBCommon represents a DBCommon Object
type DBCommon struct {
	Log *log.Logger
}

//NewDBCommon constructor
func NewDBCommon(log *log.Logger) *DBCommon {
	return &DBCommon{Log: log}
}

func (dbc *DBCommon) retornarCliente(url string, appName string) (*mongo.Client, error) {
	clientOptions := options.Client().ApplyURI("mongodb://" + url).SetConnectTimeout(10 * time.Second)
	clientOptions.SetAppName(appName)
	clientOptions.SetMaxConnIdleTime(15 * time.Second)
	clientOptions.SetServerSelectionTimeout(10 * time.Second)
	client, erro := mongo.NewClient(clientOptions)
	if erro != nil {
		dbc.Log.Fatal("Erro ao criar o cliente de conexão com o DB", erro.Error())
		return nil, erro
	}
	erro = client.Connect(context.Background())
	if erro != nil {
		dbc.Log.Fatal("Erro ao se conectar com o DB", erro.Error())
		return nil, erro
	}
	return client, nil
}

func (dbc *DBCommon) retornarClienteSeguro(url string, authDB string, user string, password string, appName string) (*mongo.Client, error) {
	credentials := options.Credential{AuthSource: authDB, Username: user, Password: password}
	connectionOptions := options.Client().ApplyURI("mongodb://" + url).SetAppName(appName).SetAuth(credentials).SetConnectTimeout(5 * time.Second)
	connectionOptions.SetMaxConnIdleTime(15 * time.Second)
	connectionOptions.SetServerSelectionTimeout(10 * time.Second)
	client, erro := mongo.NewClient(connectionOptions)
	if erro != nil {
		dbc.Log.Fatal("Erro ao criar o cliente de conexão com o DB", erro.Error())
		return nil, erro
	}
	erro = client.Connect(context.Background())
	if erro != nil {
		dbc.Log.Fatal("Erro ao se conectar com o DB", erro.Error())
		return nil, erro
	}
	return client, nil
}

func (dbc *DBCommon) retornarClienteSeguroMongoAtlas(url string, user string, password string, db string, appName string) (*mongo.Client, error) {
	connectionURL := fmt.Sprintf("mongodb+srv://%s:%s@tu-8pfjk.mongodb.net/%s?retryWrites=true&w=majority", user, password, db)
	connectionOptions := options.Client().ApplyURI(connectionURL)
	connectionOptions.SetAppName(appName).SetConnectTimeout(10 * time.Second)
	connectionOptions.SetMaxConnIdleTime(15 * time.Second)
	connectionOptions.SetServerSelectionTimeout(10 * time.Second)
	client, erro := mongo.NewClient(connectionOptions)
	if erro != nil {
		dbc.Log.Fatal("Erro ao criar o cliente de conexão com o DB", erro.Error())
		return nil, erro
	}
	erro = client.Connect(context.Background())
	if erro != nil {
		dbc.Log.Fatal("Erro ao se conectar com o DB", erro.Error())
		return nil, erro
	}
	return client, nil
}

func (dbc *DBCommon) total(nomeDB string, nomeColecao string, client *mongo.Client, filtro interface{}) (int64, error) {
	collection := client.Database(nomeDB).Collection(nomeColecao)
	total, erro := collection.CountDocuments(context.TODO(), filtro)
	return total, erro
}

func (dbc *DBCommon) deletarPeloID(nomeDB string, nomeColecao string, client *mongo.Client, insertedID interface{}) (*mongo.DeleteResult, error) {
	collection := client.Database(nomeDB).Collection(nomeColecao)
	filtro := bson.M{"_id": insertedID}
	deleteResult, erro := collection.DeleteOne(context.TODO(), filtro)
	return deleteResult, erro
}

func (dbc *DBCommon) deletar(nomeDB string, nomeColecao string, client *mongo.Client, filtro interface{}) (*mongo.DeleteResult, error) {
	collection := client.Database(nomeDB).Collection(nomeColecao)
	deleteResult, erro := collection.DeleteOne(context.TODO(), filtro)
	return deleteResult, erro
}

func (dbc *DBCommon) atualizarPeloID(nomeDB string, nomeColecao string, client *mongo.Client, insertedID interface{}, campoAtualizado interface{}) (*mongo.UpdateResult, error) {
	collection := client.Database(nomeDB).Collection(nomeColecao)
	atualizacao := bson.D{{Key: "$set", Value: campoAtualizado}}
	filtro := bson.M{"_id": insertedID}
	updateResult, erro := collection.UpdateOne(context.TODO(), filtro, atualizacao)
	return updateResult, erro
}

func (dbc *DBCommon) atualizar(nomeDB string, nomeColecao string, client *mongo.Client, filtro interface{}, campoAtualizado interface{}) (*mongo.UpdateResult, error) {
	collection := client.Database(nomeDB).Collection(nomeColecao)
	atualizacao := bson.D{{Key: "$set", Value: campoAtualizado}}
	updateResult, erro := collection.UpdateOne(context.TODO(), filtro, atualizacao)
	return updateResult, erro
}

func (dbc *DBCommon) adicionar(ctx context.Context, nomeDB string, nomeColecao string, documento interface{}, client *mongo.Client) (*mongo.InsertOneResult, error) {
	collection := client.Database(nomeDB).Collection(nomeColecao)
	c := context.TODO()
	insertOneResult, erro := collection.InsertOne(c, documento)
	return insertOneResult, erro
}

func (dbc *DBCommon) adicionarVarios(tx context.Context, nomeDB string, nomeColecao string, documentos []interface{}, client *mongo.Client) (*mongo.InsertManyResult, error) {
	collection := client.Database(nomeDB).Collection(nomeColecao)
	insertManyResult, erro := collection.InsertMany(context.TODO(), documentos)
	return insertManyResult, erro
}

func (dbc *DBCommon) retornarUm(nomeDB string, nomeColecao string, modelo interface{}, client *mongo.Client,
	filtro bson.M, findOption *options.FindOneOptions) error {
	collection := client.Database(nomeDB).Collection(nomeColecao)
	a := collection.FindOne(context.TODO(), filtro, findOption)
	erro := a.Decode(modelo)
	return erro
}

func (dbc *DBCommon) retornarTodos(ctx context.Context, nomeDB string,
	nomeColecao string, modelo interface{}, client *mongo.Client, filtro bson.M) (interface{}, error) {

	collection := client.Database(nomeDB).Collection(nomeColecao)
	cur, erro := collection.Find(context.TODO(), filtro)
	if erro != nil {
		return nil, erro
	}
	rv := reflect.ValueOf(modelo).Elem()
	sv := rv.Slice(0, rv.Cap())

	for cur.Next(context.Background()) {
		pev := reflect.New(sv.Type().Elem())
		if erro := cur.Decode(pev.Interface()); erro != nil {
			return nil, erro
		}

		sv = reflect.Append(sv, pev.Elem())
	}

	rv.Set(sv)
	return cur.Err(), erro
}
