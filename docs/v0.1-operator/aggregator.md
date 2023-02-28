# Aggregator

```shell
HOST_SETTINGS_DB_DIR=
HOST_SETTINGS_LOG_DIR=
```

0. Migration

If Aggregation service is being launched for the first time, you need to migrate initial settings first.

```shell
yarn cli migrate \
    --force \
    --migrationsPath src/cli/orakl-cli/migrations/
```

1. Setup listener

Aggregator node has to be able to catch information about new rounds.
You will need to know address of aggregator (`Aggregator`).
Emitted event is called `NewRound`.

```shell
yarn cli listener insert \
    --service Aggregator \
    --chain ${chain} \
    --address ${aggregatorAddress} \
    --eventName NewRound
```

2. Setup adapter & aggregator

In the example settings below, we are using `ETH/USD` data feed.

```shell
yarn cli adapter insert
    --chain ${chain} \
    --file-path ${adapterFilePath}

yarn cli aggregator insert \
    --chain ${chain} \
    --file-path ${aggregatorFilePath} \
    --adapter ${adapterId}
```

3. Launch

```shell
yarn start:listener:aggregator
yarn start:worker:aggregator
yarn start:reporter:aggregator
```
