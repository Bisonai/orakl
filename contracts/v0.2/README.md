# Orakl Network Contracts v0.2

## Prerequisities

Install Foundry from [here](https://book.getfoundry.sh/getting-started/installation), or by executing the command below.

```shell
curl -L https://foundry.paradigm.xyz | bash
```

Running `foundryup` by itself will install the latest (nightly) precompiled binaries: `forge`, `cast`, `anvil`, and `chisel`

## Build

```shell
forge build
```

## Test

```shell
forge test
```

## Format

```shell
forge fmt
```

## Deployment

1. Create `.env` from `.env.example` and fill in `PRIVATE_KEY`

```
cp .env.example .env
```

2. Deploy

```shell
forge script {ContractScriptName} --broadcast --rpc-url {RPC}
```

- ex

```shell
forge script DeployFull --broadcast --rpc-url http://localhost:8545
```

## Utility Scripts

### Generate Migration from Orakl Config

Following command will generate migration file for whole deployment in `./migration/${chain}/SubmissionProxy/${dateTime}_deploy.json`

```shell
node ./script/generate-migration-from-config.cjs --chain test
```

### Generated Collected Addresses

Following command collects deployed addresses into json files `datafeeds-addresses.json` and `others-addresses.json`. Stores in ./addresses path

```shell
node ./script/collect-addresses.cjs
```

## Migration Examples

### `./migration/{CHAIN(local/baobab/cypress)}/Feed/{migrationFile}.json`

- Deploy `Feed` & `FeedProxy` Contracts

```json
{
  "deploy": {
    "submitter": "0xa195bE68Bd37EBFfB056279Dc3d236fAa6F23670",
    "feedNames": ["ADA-USDT", "BTC-USDT"]
  }
}
```

- Update Submitter of `Feed` Contract

```json
{
  "updateSubmitter": {
    "submitter": "0xa195bE68Bd37EBFfB056279Dc3d236fAa6F23670",
    "feedAddresses": [
      "0xc765f5ed9abb26349054020feea04f955a5cb1ec",
      "0x9bb8f7b9f08ecc75aba62ba25d7b3f46fce79745"
    ]
  }
}
```

- Propose FeedProxy new Feeds

```json
{
  "proposeFeeds": [
    {
      "feedProxyAddress": "0x3aa5ebb10dc797cac828524e59a333d0a371443c",
      "feedAddress": "0x68b1d87f95878fe05b998f19b66f4baba5de1aed"
    },
    {
      "feedProxyAddress": "0x59b670e9fa9d0a427751af201d676719a970857b",
      "feedAddress": "0xc6e7df5e7b4f2a278906862b61205850344d4e7d"
    }
  ]
}
```

- Confirm FeedProxy new Feeds

```json
{
  "confirmFeeds": [
    {
      "feedProxyAddress": "0x3aa5ebb10dc797cac828524e59a333d0a371443c",
      "feedAddress": "0x68b1d87f95878fe05b998f19b66f4baba5de1aed"
    },
    {
      "feedProxyAddress": "0x59b670e9fa9d0a427751af201d676719a970857b",
      "feedAddress": "0xc6e7df5e7b4f2a278906862b61205850344d4e7d"
    }
  ]
}
```

### `./migration/{CHAIN(local/baobab/cypress)}/FeedRouter/{migrationFile}.json`

- Deploy `FeedRouter`

```json
{
  "deploy": {}
}
```

- Update proxies in `FeedRouter`

```json
{
  "address": "0x1ac6cd893eddb6cac15e5a9fc549335b8b449015",
  "updateProxyBulk": [
    {
      "feedName": "BTC-USDT",
      "proxyAddress": "0x50c23983ea26f30d368da5b257001ee3ddf9a539"
    },
    {
      "feedName": "KLAY-USDT",
      "proxyAddress": "0xd07bd0bcd3a8fa1087430b1be457e05c4a412a4b"
    }
  ]
}
```

### `./migration/{CHAIN(local/baobab/cypress)}/SubmissionProxy/{migrationFile}.json`

- Deploy `SubmissionProxy` and Register Oracle

```json
{
  "deploy": {},
  "addOracle": {
    "oracles": ["0x50c23983ea26f30d368da5b257001ee3ddf9a539"]
  }
}
```

- Update Configuration of `SubmissionProxy`

```json
{
  "address": "0x1ac6cd893eddb6cac15e5a9fc549335b8b449015",
  "setMaxSubmission": 120,
  "setDataFreshness": 2,
  "setExpirationPeriod": 2592000,
  "setDefaultProofThreshold": 80,
  "setProofThreshold": [
    {
      "name": "BTC-USDT",
      "threshold": 60
    }
  ],
  "addOracle": {
    "oracles": ["0x50c23983ea26f30d368da5b257001ee3ddf9a539"]
  },
  "removeOracle": {
    "oracles": ["0xd07bd0bcd3a8fa1087430b1be457e05c4a412a4b"]
  },
  "updateFeed": [
    {
      "name": "BTC-USDT",
      "feedAddress": "0xB7f8BC63BbcaD18155201308C8f3540b07f84F5e"
    },
    {
      "name": "ETH-USDT",
      "feedAddress": "0x0DCd1Bf9A1b36cE34237eEaFef220932846BCD82"
    }
  ],
  "removeFeed": {
    "feedNames": ["BNB-USDT", "PEPE-USDT"]
  }
}
```

- Deploy `SubmissionProxy` with `Feed` and `FeedProxy` contracts

```json
{
  "deploy": {},
  "deployFeed": {
    "feedNames": ["ADA-USDT", "ATOM-USDT"]
  }
}
```
