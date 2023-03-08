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

## Configure Orakl Network Fetcher

```shell
yarn cli chain insert --name localhost
yarn cli adapter insert --file-path adapter/btc-usdt.adapter.json

# adapterHash=0xd6fbe30bd6249b3093ee065496115e5736bbe760cadfc85598ef27eb4739a849
yarn cli aggregator insert --file-path aggregator/localhost/btc-usdt.aggregator.json --chain localhost
```

## Start data collection and aggregation

```shell
yarn cli fetcher start --id 0xd6fbe30bd6249b3093ee065496115e5736bbe760cadfc85598ef27eb4739a849 --chain localhost
```

## Stop data collection and aggregation

```shell
yarn cli fetcher stop --id 0xd6fbe30bd6249b3093ee065496115e5736bbe760cadfc85598ef27eb4739a849 --chain localhost
```
