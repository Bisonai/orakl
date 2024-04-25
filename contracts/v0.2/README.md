# Orakl Network Contracts v0.2

## Prerequisities

Install Foundry by following description at https://book.getfoundry.sh/getting-started/installation or by executing the command below.

```shell
curl -L https://foundry.paradigm.xyz | bash

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

## Migration Examples

### Deploy `Feed` & `FeedProxy` Contracts

```
{
  "Feed": {
    "deploy": {
      "submitter": "0xa195bE68Bd37EBFfB056279Dc3d236fAa6F23670",
      "count": 2,
      "feeds": [
        { "decimals": 8, "description": "ADA-USDT" },
        { "decimals": 8, "description": "ATOM-USDT" }
      ]
    }
  }
}
```

### Update Submitter of `Feed` Contract

```
{
  "Feed": {
    "updateSubmitter": {
      "submitter": "0xa195bE68Bd37EBFfB056279Dc3d236fAa6F23670",
      "count": 2,
      "feeds": [
        "0xc765f5ed9abb26349054020feea04f955a5cb1ec",
        "0x9bb8f7b9f08ecc75aba62ba25d7b3f46fce79745"
      ]
    }
  }
}
```

### Deploy `FeedRouter`

```
{
  "FeedRouter": {
    "deploy": {}
  }
}

```

### Update proxies in `FeedRouter`

```
{
  "FeedRouter": {
    "address": "0x1ac6cd893eddb6cac15e5a9fc549335b8b449015",
    "updateProxyBulk": {
      "count": 2,
      "proxies": [
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
  }
}
```

### Deploy `SubmissionProxy` and Register Oracle

```
{
  "SubmissionProxy": {
    "deploy": {},
    "addOracle": {
      "count": 1,
      "oracles": ["0x50c23983ea26f30d368da5b257001ee3ddf9a539"]
    },
  }
}
```

### Update Configuration of `SubmissionProxy`

```
{
  "SubmissionProxy": {
    "address": "0x1ac6cd893eddb6cac15e5a9fc549335b8b449015",
    "setMaxSubmission": 120,
    "setDataFreshness": 2,
    "setExpirationPeriod": 2592000,
    "setDefaultProofThreshold": 80,
    "setProofThreshold": {
      "count": 1,
      "thresholds": [
        {
          "feed": "0xd07bd0bcd3a8fa1087430b1be457e05c4a412a4b",
          "threshold": 60
        }
      ]
    },
    "addOracle": {
      "count": 1,
      "oracles": ["0x50c23983ea26f30d368da5b257001ee3ddf9a539"]
    },
    "removeOracle": {
      "count": 1,
      "oracles": ["0xd07bd0bcd3a8fa1087430b1be457e05c4a412a4b"]
    }
  }
}
```

### Deploy `SubmissionProxy` with `Feed` and `FeedProxy` contracts

```
{
  "SubmissionProxy": {
    "deploy": {},
    "deployFeed": {
      "count": 2,
      "feeds": [
        { "decimals": 8, "description": "ADA-USDT" },
        { "decimals": 8, "description": "ATOM-USDT" }
      ]
    }
  }
}
```
