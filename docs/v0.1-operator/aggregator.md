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
    --chain baobab \
    --file-path adapter/eth-usd.adapter.json

yarn cli aggregator insert \
    --chain baobab \
    --file-path aggregator/baobab/eth-usd.aggregator.json \
    --adapter 0x7e6552824ce107ab0d6e4266ba6b93f0afe5aa576a491364fc01881a34ddb12b
```

3. Launch

```shell
yarn start:listener:aggregator
yarn start:worker:aggregator
yarn start:reporter:aggregator
```
