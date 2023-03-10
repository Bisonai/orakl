# Orakl Network API

## Installation

```shell
yarn install
```

## Settings

Orakl Network API requires to set the following environment variables.

* `DATABASE_URL`
* `APP_PORT`

You can copy them from `.env.example` to `.env` and fill the appropriate values.

```shell
cp .env.example .env
```

## Local development

```shell
brew install postgresql@14
brew services start postgresql@14
```

```shell
createdb orakl
#dropdb orakl
```

## Prisma

```shell
npx prisma format
npx prisma migrate dev --name init
```

## New endpoint

```shell
nest g resource name
```

## Testing

```shell
createdb orakl-test
DATABASE_URL="postgresql://${USER}@localhost:5432/orakl-test?schema=public" npx prisma migrate dev --name init
DATABASE_URL="postgresql://${USER}@localhost:5432/orakl-test?schema=public" yarn test
dropdb orakl-test
```

## Running the app

```shell
# development
yarn run start

# watch mode
yarn run start:dev

# production mode
yarn run start:prod
```

Go to http://localhost:3000/api

## Tests

```shell
# unit tests
yarn test

# e2e tests
yarn test:e2e

# test coverage
yarn test:cov
```

## Endpoints

## Health

```shell
GET http://localhost:3000/health
```

### Open API (Swagger)

```shell
GET http://localhost:3000/docs
```

### List data feeds

```shell
GET http://localhost:3000/api/v1/feed
```

### List all adapters

```shell
GET http://localhost:3000/api/v1/adapter
```

### List all aggregators

```shell
GET http://localhost:3000/api/v1/aggregator
```

## How to use?

1. Insert `Chain`s (should be done only once, can be included in migration file)
2. Insert `Adapter` (initial settings)
3. Insert `Aggregator` (initial settings)
3. Insert `Data` (during regular data fetching with Orakl Network Fetcher)
