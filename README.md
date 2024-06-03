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

### Data feed

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

3. Docker Compose Down
   Close all related containers.

```bash
docker-compose -f docker-compose.local-data-feed.yaml down -v
```

### VRF / Request-Response

1. Docker Compose Build
   Builds all required images for docker-compose.

```bash
docker-compose -f docker-compose.local-core.yaml build
```

2. Docker Compose Up
   Runs all required images to run datafeed locally.

```bash
SERVICE=rr docker-compose -f docker-compose.local-core.yaml up --force-recreate
```

3. Docker Compose Down
   Close all related containers.

```bash
docker-compose -f docker-compose.local-core.yaml down -v
```

Replace `SERVICE` with whichever service you'd like to run. The options are `vrf` and `rr` which represent VRF and Request-Response services respectively.

Here is what happens after the above command is run:

- `api`, `postgres`, `redis`, and `json-rpc` services will start as separate docker containers
- `postgres` will get populated with necessary data:
  - chains
  - services
  - vrf keys in case if the service is vrf
  - listener (after contracts are deployed)
  - reporter (after contracts are deployed)
- migration files in `contracts/v0.1/migration/` get updated with provided keys and other values
- relevant coordinator and prepayment contracts get deployed

Keep in mind that you'll need the [keyHash](/dockerfiles/local-vrf-rr/envs/vrf-keys.json) value for VRF consumer and update it in `vrf-consumer/scripts/utils.ts`

You can spin up the listener, worker, and reporter services from [core](../../core/) and make requests to VRF or Request-Response consumers after deploying consumer contracts.

### Notes

- The current automation is not designed to run both VRF and Request-Response services.
- Therefore, every time a new service (VRF or Request-Response) is started, all the running containers related to `core` will be recreated, meaning you'll lose all changes in those containers
