# Orakl Network API

## Installation

```shell
yarn install
```

## Local development

```shell
brew install postgresql@14
brew services start postgresql@14
```

```shell
createdb orakl
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
yarn run test

# e2e tests
yarn run test:e2e

# test coverage
yarn run test:cov
```

## Endpoints

## Health

GET http://localhost:3000/health

### Open API (Swagger)

GET http://localhost:3000/docs

### List data feeds

GET http://localhost:3000/api/v1/feed
