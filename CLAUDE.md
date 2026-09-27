# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## How to work in this repo

When a step doesn't need my input, keep going. Put status notes in the same message as your next action. Stop and ask only when you can't continue without me, or before anything destructive: deleting data, force-pushing, or changing anything outside this repository.

A change is done when `go vet ./...` and `go test ./...` pass against the local MongoDB described below. If you couldn't run the tests (for example, no MongoDB available), say so instead of reporting the change as verified.

## Overview

`github.com/eduardohitek/golang-mongo-common/v2` is a small Go library (package `common`) of helper functions around the official MongoDB Go driver v2 (`go.mongodb.org/mongo-driver/v2`). Other projects import it; it is not an executable. The code lives in `common.go`, the package doc (including the v1 → v2 migration guide) in `doc.go`, tests in `common_test.go`, and pkg.go.dev examples in `example_test.go`. v1 of the library (driver v1.17.x) is maintained only for critical fixes on the `v1` branch.

## Commands

```sh
go build ./...                                  # compile
go vet ./...                                    # static checks
go test ./...                                   # run all tests
go test -run Test_CreateIndex -v ./...          # run a single test
go mod tidy                                     # after dependency changes
```

Most tests are integration tests. They connect through `getClient(t)` and need a MongoDB instance on `localhost:27017` with no authentication. Each test uses its own collection in the `test` database (`newCollection(t, client)`) and drops it on cleanup. A quick way to get one:

```sh
docker run -d --name mongo-test -p 27017:27017 mongo:8.0
```

The examples in `example_test.go` have no `// Output:` line, so they are compiled but never run and don't need a database.

## Architecture and conventions

- **Client constructors** (`ReturnClient`, `ReturnAuthenticatedClient`, `ReturnAuthenticatedClientMongoAtlas`):
  - they take `ctx` plus variadic `ClientOption` values (`WithLogger`, `WithTimeouts`), which fill a private `clientConfig`;
  - `setClientOptions` applies the defaults, and `connect` calls `mongo.Connect` and then `Ping` with `ctx`, disconnecting if the ping fails;
  - they return errors and never call `log.Fatal`;
  - `atlasURI` builds the `mongodb+srv` URI with `url.UserPassword`, so credentials are escaped.
- **Command logging**: `WithLogger` attaches an `event.CommandMonitor` that logs every command except `endSessions` at Debug level via `slog`.
- **CRUD helpers**:
  - all have the shape `(ctx, client *mongo.Client, dbName, collectionName, ..., opts ...options.Lister[...])`;
  - they resolve the collection on every call and forward the driver's options;
  - filters and ids are `any`, and the update helpers wrap the given value in `$set`;
  - driver errors are returned unwrapped.
- **Reads are generic**: `FindOne[T]` returns `(T, error)` and `mongo.ErrNoDocuments` when nothing matches; `FindAll[T]` returns a non-nil `[]T`.
- **Driver v2 quirk**: when decoding into `bson.M`/`any`, nested documents come back as `bson.D`.
- **Migration docs are part of the API**: every exported function has a godoc block `Migrating from v1:` with the v1 signature, the v2 signature and, when the call shape changed, a before/after one-liner. Keep it when editing a function, and keep `doc.go`, the README section "Migrando da v1 para a v2" and its signature table in sync with any signature change.
- The public API is consumed by other modules, so treat exported signature changes as breaking.
