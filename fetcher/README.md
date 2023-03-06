# Orakl Network Fetcher

Orakl Network Fetcher collects regularly data defined through aggregators defined within [Orakl Network API](https://github.com/Bisonai/orakl/tree/master/api).

## Installation

```shell
yarn install
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
