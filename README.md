# Orakl Network

This repository is split to [on-chain](contracts) and [off-chain](core) oracle implementation.

You can learn more about the Orakl Network from [documentation](https://orakl-network.gitbook.io).

# Test Data Feed Locally Using Docker

Run local data feed connected to testnet.

## Steps run through docker compose

- Deploy contracts in testnet(baobab)
- Run postgres & redis
- Run orakl-api & orakl-delegator
- Insert deployed data feed & set delegator fee payer
- Run listener, worker, reporter, and fetcher
- Activate inserted data feed

## Prerequisites

1. Docker

```bash
brew install docker
brew install docker-compose
```

2. Env setup

Nearly everything is already setup, but there are two variables that should be set manually in following env files:

- `dockerfiles/local-data-feed/envs/.contracts.env`

```
MNEMONIC="{MNEMONIC for contract deployer wallet}"
```

- `dockerfiles/local-data-feed/envs/.cli.env`

```
DELEGATOR_REPORTER_PK={private key for delegator fee payer}
```

## Run docker

1. Docker Compose Build
   Builds all required images for docker-compose.

```bash
docker-compose -f docker-compose.local-data-feed.yaml build
```

2. Docker Compose Up
   Runs all required images to run datafeed locally.

```bash
docker-compose -f docker-compose.local-data-feed.yaml up
```

2. Docker Compose Down
   Close all related containers.

```bash
docker-compose -f docker-compose.local-data-feed.yaml down -v
```


## Image Tag List

- **** <br> *`PR`*: Hotfix add missing return statement on switch default <br><br> 
- **** <br> *`PR`*: Hotfix add missing return statement on switch default <br><br> 
- **** <br> *`PR`*: Hotfix add missing return statement on switch default <br><br> 
- **** <br> *`PR`*: Hotfix add missing return statement on switch default <br><br> 
