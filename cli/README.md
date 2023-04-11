# Orakl Network CLI

The Orakl Network CLI is a tool to configure and manage the [Orakl Network](https://orakl.network).
To learn more about the Orakl Network CLI, visit [Orakl Network CLI documentation page](https://orakl-network.gitbook.io/docs/orakl-network-cli/introduction).

## Development

```shell
yarn install
yarn build
```

## Test

```shell
yarn test
```

## Lint

```shell
yarn lint
```

## Environment Variables

The Orakl Network CLI needs to communicate with other Orakl Network services (**Orakl Network API**, **Orakl Network Fetcher**, and **Orakl Network Delegator**) to function properly.
The services are expected to be launched before using the Orakl Network CLI.
The Orakl Network CLI tries to connect to the required services with URL environment variables.

- `ORAKL_NETWORK_API_URL`
- `ORAKL_NETWORK_FETCHER_URL`
- `ORAKL_NETWORK_DELEGATOR_URL`

## Publishing

The `@bisonai/orakl-cli` package is published through [Github Actions pipeline](https://github.com/Bisonai/orakl/blob/master/.github/workflows/cli.build+publish.yaml) when the version the package specified in `package.json` changes.

The package is published at [NPM registry](https://www.npmjs.com/package/@bisonai/orakl-cli).

## License

MIT

## How to Run Delegator throught Cli

Insert New Organization

```shell
 yarn cli delegator organizationInsert --name BisonAI
```

List Organizations

```shell
 yarn cli delegator organizationList
```

```json
[{ "id": "1", "name": "BisonAI" }]
```

Insert New Reporter

```shell
yarn cli delegator reporterInsert \
      --address "0x0000000000000000000000000000000000000000" \
      --organizationId 1
```

List Reporters

```shell
yarn cli delegator reporterList
```

Insert New Contract

```shell
yarn cli delegator contractInsert \
      --address "0x0000000000000000000000000000000000000001"
```

List Contracts

```shell
yarn cli delegator contractList
```

Insert New Function

```shell
yarn cli delegator functionInsert \
      --name "sendRequest()" \
      --contractId 1
```

List Functions

```shell
yarn cli delegator functionList
```

Connect contract with reporter

```shell
yarn cli delegator contractConnect \
      --contractId 1 \
      --reporterId 1
```

```shell
yarn cli delegator reporterList
```
