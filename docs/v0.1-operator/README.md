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

### Loading settings from file

Node settings as described in previous section can be also predefined in JSON file and set all with single command.

Below you can see the contents of a JSON file (`settings.json`) containing the same settings as in the section above.

```
{
  "PROVIDER_URL": "https://api.baobab.klaytn.net:8651",
  "HEALTH_CHECK_PORT": "8888",
  "REDIS_HOST": "localhost",
  "REDIS_PORT": "6379",
  "PRIVATE_KEY": "0x...",
  "PUBLIC_KEY": "0x...",
  "LOCAL_AGGREGATOR": "MEDIAN",
  "LISTENER_DELAY": "500"
}
```

To load all settings with CLI at once, run the code below.

```shell
yarn cli kv insertMany \
    --file-path settings.json \
    --chain ${chain}
```

## Guides

* [Verifiable Random Function (VRF)](vrf.md)
* [Aggregator](aggregator.md)
