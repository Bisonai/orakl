# v0.1 for operators

```shell
nvm use v18.12.1
yarn add @bisonai-cic/icn-core
```

## Prerequisities

All settings are store in local database.
Run following command to create database and required tables with default values for `localhost` network.

```shell
yarn cli migrate --force
```

WARNING: If you run this command later, after you have modified any settings, all settings will be wiped out.

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

## Verifiable Random Function (VRF)

1. Generate VRF keys

To provide VRF service, one must generate VRF private and public keys.
All keys can be generated with the command below

```shell
yarn keygen
```

The output format of command is shown below

```shell
SK=
PK=
PK_X=
PK_Y=
KEY_HASH=
```

The generated `KEY_HASH` uniquely represents your VRF keys without exposing them.
This hash must be registered using `VRFCoordinator.registerProvingKey`.
Please share with Bisonai this `KEY_HASH` and your `PUBLIC_KEY` that corresponds to your wallet address.

2. Setup VRF keys

The keys generated in previous step are supplied to `yarn cli vrf insert` command.
Parameter `--chain` corresponds to the name network to which VRF keys will be associated.

```shell
yarn cli vrf insert \
    --chain baobab \
    --pk [PK] \
    --sk [SK] \
    --pk_x [PK_X] \
    --pk_y [PK_Y]
```

3. Launch `listener`, `worker` and `reporter`

TODO update with Docker compose launch.

```shell
yarn start:listener:vrf
yarn start:worker:vrf
yarn start:reporter:vrf
```
