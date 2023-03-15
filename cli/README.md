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

The Orakl Network CLI needs to communicate with other Orakl Network services (**Orakl Network API** and **Orakl Network Fetcher**) to function properly.
The services are expected to be launched before using the Orakl Network CLI.
The Orakl Network CLI tries to connect to the required services with URL environment variables.

* `ORAKL_NETWORK_API_URL`
* `ORAKL_NETWORK_FETCHER_URL`

## Publishing

The Orakl Network CLI (`@bisonai/orakl-cli`) is published through [Github Actions pipeline](https://github.com/Bisonai/orakl/blob/master/.github/workflows/cli.build+publish.yaml) when package version specified in `package.json` changes.

The package is published at [NPM registry](https://www.npmjs.com/package/@bisonai/orakl-cli).

## License

MIT
