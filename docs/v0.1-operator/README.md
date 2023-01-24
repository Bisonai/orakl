# v0.1 for operators

Use node version `18.12.1`

```shell
nvm use v18.12.1
```

Clone, build and setup

```shelll
git clone https://github.com/bisonai-cic/ICN
cd ICN/core

# FIXME
export NPM_TOKEN=
yarn install
yarn build

touch .env
echo "NODE_ENV=production" >> .env
echo "CHAIN=localhost" >> .env
```

## Docker

```shell
docker compose -f docker-compose.build.yaml build
```

## Prerequisities

All settings are stored in a local database.
Run following command to create the database and all required tables with default values for `localhost` network.

```shell
yarn cli migrate --force
```

WARNING: If you run this command after you have modified any settings, all previous settings will be wiped out.

## Settings

This section described all necessary settings that has to be set before launching a node.
Every settings is tied to a chain, therefore we recommend to set chain name to environemnt variable and reuse it later.
In the example below, we use `baobab` which is Klaytn's test net.

```shell
export chain=baobab
```

```shell
# General
yarn cli kv insert --chain ${chain} --key PROVIDER_URL      --value https://api.baobab.klaytn.net:8651
yarn cli kv insert --chain ${chain} --key HEALTH_CHECK_PORT --value 8888

# TODO update with docker compose settings
yarn cli kv insert --chain ${chain} --key REDIS_HOST        --value localhost
yarn cli kv insert --chain ${chain} --key REDIS_PORT        --value 6379

# Reporter wallet
yarn cli kv insert --chain ${chain} --key PRIVATE_KEY       --value 0x...
yarn cli kv insert --chain ${chain} --key PUBLIC_KEY        --value 0x...

# Aggregator
yarn cli kv insert --chain ${chain} --key LOCAL_AGGREGATOR  --value MEDIAN

# Listener
yarn cli kv insert --chain ${chain} --key LISTENER_DELAY    --value 500
```

## Guides

* [Verifiable Random Function (VRF)](vrf.md)
* [Aggregator](aggregator.md)
