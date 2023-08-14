# Orakl Network L2-Api

## Installation

```shell
yarn install
```

## Settings

Orakl Network L2 Config API requires to set the following environment variables.

- `DATABASE_URL`
- `APP_PORT`

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
createdb orakl-l2-api
#dropdb orakl-l2-api
```

## Prisma

```shell
npx prisma format
npx prisma migrate dev --name init
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

## Test

```bash
# unit tests
$ yarn run test

# e2e tests
$ yarn run test:e2e

# test coverage
$ yarn run test:cov
```

## Endpoints

### Health

```shell
GET http://localhost:3000/api/v1
```

### Open API (Swagger)

```shell
GET http://localhost:3000/docs
```
