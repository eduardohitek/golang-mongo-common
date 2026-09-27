# Golang Mongo Common

![GitHub repo size](https://img.shields.io/github/repo-size/eduardohitek/golang-mongo-common)
![GitHub Repo stars](https://img.shields.io/github/stars/eduardohitek/golang-mongo-common)
![GitHub issues](https://img.shields.io/github/issues/eduardohitek/golang-mongo-common)


> Provide some Helpers functions for MongoDB use with Golang.

Built on the official driver [`go.mongodb.org/mongo-driver/v2`](https://pkg.go.dev/go.mongodb.org/mongo-driver/v2). Requires Go 1.25+.

## Instalação

```sh
go get github.com/eduardohitek/golang-mongo-common/v2
```

## Uso

```go
import (
	"context"
	"log"

	common "github.com/eduardohitek/golang-mongo-common/v2"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type User struct {
	ID   bson.ObjectID `bson:"_id,omitempty"`
	Name string        `bson:"name"`
}

func main() {
	ctx := context.Background()
	client, err := common.ReturnClient(ctx, "localhost:27017", "my-app")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	_, err = common.InsertOne(ctx, client, "app", "users", User{Name: "Ana"})
	if err != nil {
		log.Fatal(err)
	}

	users, err := common.FindAll[User](ctx, client, "app", "users", bson.M{})
	if err != nil {
		log.Fatal(err)
	}
	log.Println(len(users))
}
```

Para logar os comandos enviados ao MongoDB, passe `common.WithLogger(logger)` com um `*slog.Logger` em nível Debug. Para mudar os timeouts padrão, use `common.WithTimeouts(...)`.

## Migrando da v1 para a v2

A v2 usa o driver v2 do MongoDB e muda as assinaturas das funções. Cada função exportada tem, no godoc, um bloco `Migrating from v1:` com a assinatura antiga, a nova e um exemplo antes/depois. `go doc github.com/eduardohitek/golang-mongo-common/v2` mostra o resumo.

### Passo a passo

1. **Go 1.25+:** suba a diretiva `go` do seu `go.mod` para `1.25` ou mais.
2. **Dependência:** rode `go get github.com/eduardohitek/golang-mongo-common/v2@latest` e depois `go mod tidy` (que remove a v1 quando ela não for mais importada).
3. **Import da lib:** troque `github.com/eduardohitek/golang-mongo-common` por `github.com/eduardohitek/golang-mongo-common/v2`. O nome do pacote continua `common`.
4. **Import do driver:** migre também o seu código para o driver v2, trocando `go.mongodb.org/mongo-driver/...` por `go.mongodb.org/mongo-driver/v2/...`. Mudanças mais comuns:
   - `primitive.ObjectID` → `bson.ObjectID` (o pacote `primitive` não existe mais; `primitive.D`/`primitive.M` viram `bson.D`/`bson.M`);
   - options são builders: `options.FindOne().SetProjection(...)`;
   - ao decodificar em `bson.M` ou `any`, documentos aninhados vêm como `bson.D`, e não mais como `bson.M`.

   O resto está no [guia oficial de migração do driver](https://www.mongodb.com/docs/drivers/go/current/reference/upgrade/).
5. **Context:** passe um `ctx context.Context` como primeiro argumento em todas as chamadas.
6. **Construtores:**
   - remova o último argumento `createMonitor`. Se era `true`, passe `common.WithLogger(logger)`; se era `false`, não passe nada;
   - trate o erro retornado. A v1 encerrava o processo com `log.Fatal`; a v2 retorna o erro;
   - os construtores agora fazem `Ping`, então um banco inacessível é detectado já na criação do client;
   - no construtor do Atlas, se você escapava a senha por conta própria, deixe de escapar: a lib já faz isso.
7. **`FindOne` / `FindAll`:** remova o argumento `model` e a type assertion e passe o tipo como parâmetro:
   ```go
   // v1
   res, err := common.FindOne(client, "app", "users", &User{}, bson.M{"name": "Ana"}, nil)
   user := res.(*User)
   // v2
   user, err := common.FindOne[User](ctx, client, "app", "users", bson.M{"name": "Ana"})

   // v1
   res, err := common.FindAll(client, "app", "users", []User{}, bson.M{})
   users := res.([]User)
   // v2 (T é o tipo do elemento, não da slice)
   users, err := common.FindAll[User](ctx, client, "app", "users", bson.M{})
   ```
   O `FindOne` v2 retorna o valor (`User`), não um ponteiro. Opções como projeção são passadas no final: `common.FindOne[User](ctx, c, db, coll, filter, options.FindOne().SetProjection(p))`.
8. **Documento não encontrado:** use `errors.Is(err, mongo.ErrNoDocuments)`.
9. **Valide:** rode `go build ./... && go vet ./...` e os testes do seu projeto. O compilador aponta cada chamada que ainda falta migrar.

### Tabela de assinaturas

| Função | v1 | v2 |
|---|---|---|
| `ReturnClient` | `(url, appName string, createMonitor bool)` | `(ctx, host, appName string, opts ...ClientOption)` |
| `ReturnAuthenticatedClient` | `(url, authDB, user, password, appName string, createMonitor bool)` | `(ctx, host, authDB, user, password, appName string, opts ...ClientOption)` |
| `ReturnAuthenticatedClientMongoAtlas` | `(url, user, password, db, appName string, createMonitor bool)` | `(ctx, host, user, password, db, appName string, opts ...ClientOption)` |
| `Total` | `(client, db, coll string, filter bson.M) (int64, error)` | `(ctx, client, db, coll string, filter any, opts...) (int64, error)` |
| `InsertOne` | `(client, db, coll string, model interface{})` | `(ctx, client, db, coll string, document any, opts...)` |
| `InsertMany` | `(client, db, coll string, models []interface{})` | `(ctx, client, db, coll string, documents []any, opts...)` |
| `FindOne` | `(client, db, coll string, model interface{}, filter bson.M, findOption *options.FindOneOptions) (interface{}, error)` | `FindOne[T](ctx, client, db, coll string, filter any, opts...) (T, error)` |
| `FindAll` | `(client, db, coll string, model interface{}, filter bson.M) (interface{}, error)` | `FindAll[T](ctx, client, db, coll string, filter any, opts...) ([]T, error)` |
| `UpdateByID` | `(client, db, coll string, insertedID, updatedField interface{})` | `(ctx, client, db, coll string, id, update any, opts...)` |
| `UpdateOneByFilter` | `(client, db, coll string, filter bson.M, updatedField interface{})` | `(ctx, client, db, coll string, filter, update any, opts...)` |
| `UpdateManyByFilter` | `(client, db, coll string, filter bson.M, updatedField interface{})` | `(ctx, client, db, coll string, filter, update any, opts...)` |
| `DeleteOneByID` | `(client, db, coll string, insertedID interface{})` | `(ctx, client, db, coll string, id any, opts...)` |
| `DeleteOneByFilter` | `(client, db, coll string, filter bson.M)` | `(ctx, client, db, coll string, filter any, opts...)` |
| `DeleteManyByFilter` | `(client, db, coll string, filter bson.M)` | `(ctx, client, db, coll string, filter any, opts...)` |
| `CreateIndex` | `(client, db, coll, field string, unique bool) error` | `(ctx, client, db, coll, field string, unique bool) error` |

Novos na v2: `ClientOption`, `WithLogger`, `WithTimeouts`.

A v1 continua disponível (`go get github.com/eduardohitek/golang-mongo-common@v1`) e recebe só correções críticas.

## ⚠️ License

Copyright (c) 2019-present [Eduardo Hitek](https://github.com/eduardohitek). `Golang Mongo Common` is free and open-source software licensed under the [MIT License](https://github.com/gofiber/fiber/blob/master/LICENSE).
