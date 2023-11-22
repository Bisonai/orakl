# Orakl Network Fetcher

The Orakl Network Fetcher regularly collects data defined through aggregators registered within the [Orakl Network API](https://github.com/Bisonai/orakl/tree/master/api).

## Installation

```shell
yarn install
```

## Settings

Orakl Network Fetcher requires to set the following environment variables.

- `REDIS_HOST`
- `REDIS_PORT`
- `ORAKL_NETWORK_API_URL`
- `APP_PORT`
- `CHAIN`
- `FETCHER_TYPE`
- `CONCURRENCY`

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

- `GET /` (health endpoint)
- `GET /api`
- `GET /api/v1/start/{aggregator}`
- `GET /api/v1/stop/{aggregator}`

## Test

```shell
# unit tests
yarn run test

# e2e tests
yarn run test:e2e

# test coverage
yarn run test:cov
```

## Documentation

- [Learn about how the Orakl Network Fetcher can be controlled](https://orakl-network.gitbook.io/docs/orakl-network-cli/fetcher) by the [Orakl Network CLI](https://orakl-network.gitbook.io/docs/orakl-network-cli/introduction)
- [Documentation for the Orakl Network Operators](https://orakl-network.gitbook.io/docs/node-operators-guide/orakl-network-fetcher)

## License

[MIT License](LICENSE)

## Test adapter.json with script

Check if adapter json file is working properly with script added in `src/tools/run-adapter.ts`.
Place the json file inside `tools` folder and run the script with following command.

```
// example
npx ts-node ./src/tools/run-adapter.ts ./ada-usdt.adapter.json
```
