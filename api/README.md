# Orakl Network API

## To Dos

- [ ] swagger

## Structure

- `main.go` : entrypoint to run api server
- `/utils/utils.go`: package containing utility functions
- `/{service}/route.go`: contains routes each calling its function in controller
- `/{service}/controller.go`: contains model and function referenced from endpoint
- `/{service}/queries.go`: contains query or query generator to call db
- `/tests/{service}_test.go`: contains test for each service

## Naming convention

### PascalCase

- exported(called outside package) function or variable
- struct

```go
type FeedInsertModel struct {...}
func GenerateGetListenerQuery(params map[string]string) string {...}
const (GetProxy = `SELECT * FROM proxies ORDER BY id asc;`)
```

### camelCase

- function and variables which is used within package

### CamelCase starting with Capital letter

- elements inside struct

```go
type ProxyInsertModel struct {
	Protocol int    `db:"protocol" json:"protocol" validate:"required"`
	Host     string `db:"host" json:"host" validate:"required"`
	Port     int    `db:"port" json:"port" validate:"required"`
	Location string `db:"location" json:"location"`
}
```

### Other rules

- some model starts with \_(underbar), it means that it's used within controller. Otherwise it means that its structure for request payload

```go
type ReporterInsertModel struct {} // struct taken from request body parameters
type _ReporterInsertModel struct {} // struct used when calling insert query
```

## Used libraries

### Api

- go-fiber (api framework)
- pgx (postgres client)
- gp-redis (redis client)

### DB migration tool

- go-migrate (db migration)

# How to run

## Prerequisites

### Install go

```bash
brew install go
```

### Install db

- Just as orakl-api, it requires postgres and redis

```bash
brew install postgresql
brew install redis
```

### Set .env

```bash
cp .env.example .env
```

- One thing that is different from orakl-api is when setting postgresql url, `?schema={schema}` should be `?search_path={schema}`.
- If port is not defined, api port will be 3000. Other environment variables are required.
- If `TEST_MODE` is true, some routes which aren't used in production will be accessable.

## Run

```bash
go run main.go
```

# How to run test

From root path run following command

```
go test ./tests -v
```

- `-v` is verbose option

## Run docker-compose from local environment

- Change api service's docker image into bisonai.com/orakl/apis
- And if there's `schema={}` in DB connection url in .api.env file update it into keyword `search_path={}`

# How to use DB migration tool

- This is meant for future development (ex. adding new column or table), don't run it on existing dbs

## Install golang-migrate

```bash
brew install golang-migrate
```

## Migrate commands

- Run commands from bisonai.com/orakl/api folder
- Write appropriate db connection url for each usecases
- Be careful on adding `sslmode=disbale`, if it has other option such as `?schema=public` add `&sslmode=disable` else add `?sslmode=disable`

### `migrate create`

create empty migration files with a pair of .up file and .down file

```bash
migrate create -ext sql -dir ./migrations -seq {migration_file_name}
```

### `migrate up`

```bash
migrate -database "postgres://{USER}@localhost:5432/orakl?sslmode=disable" -path ./migrations up
```

### `migrate down`

```bash
migrate -database "postgres://{USER}@localhost:5432/orakl?sslmode=disable" -path ./migrations down
```

### `migrate force`

Reference: https://github.com/golang-migrate/migrate/blob/0815e2d770003b4945a4bf86850fb92ca4b7cc5e/GETTING_STARTED.md#forcing-your-database-version

- If migration file contained an error, migrate will not let you run other migrations on the same database
- Once you know, you should force your database to a version reflecting its real state

```bash
migrate -database "postgres://{USER}@localhost:5432/orakl?sslmode=disable" -path ./migrations force ${VERSION}
```

## References

- https://gofiber.io/: go fiber
- https://github.com/jackc/pgx: pgx (postgres driver)
- https://github.com/redis/go-redis: go-redis
- https://github.com/golang/go/issues/27179: golang map doesn't preserve json key order, use json.rawMessage instead
- https://stackoverflow.com/questions/69762108/implementing-ethereum-personal-sign-eip-191-from-go-ethereum-gives-different-s: Keccak256Hash in golang
- https://github.com/golang-migrate/migrate: go-migrate
- https://github.com/golang-migrate/migrate/blob/master/database/postgres/TUTORIAL.md: Postgres migration tutorial
