# Orakl Network Fetcher

Orakl Network Fetcher collects regularly data defined through aggregators defined within [Orakl Network API](https://github.com/Bisonai/orakl/tree/master/api).

## Installation

```shell
yarn install
```

## Settings

Orakl Network Fetcher requires to set the following environment variables.

* `REDIS_HOST`
* `REDIS_PORT`
* `ORAKL_NETWORK_API_URL`
* `APP_PORT`

You can copy them from `.env.example` to `.env` and fill the appropriate values.

```shell
cp .env.example .env
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

## Endpoints

* `GET /health`
* `GET /api`
* `GET /api/v1/start/{aggregator}`
* `GET /api/v1/stop/{aggregator}`

## Test

```shell
# unit tests
yarn run test

# e2e tests
yarn run test:e2e

# test coverage
yarn run test:cov
```
