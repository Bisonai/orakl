# Aggregator

```
HOST_SETTINGS_DB_DIR=
```

1. Setup listener

Aggregator node has to be able to catch information about new rounds.
You will need to know address of aggregator (`Aggregator`).
Emitted event is called `NewRound`.

```
yarn cli listener insert \
    --service Aggregator \
    --chain ${chain} \
    --address ${aggregatorAddress} \
    --eventName NewRound
```

2. Setup adapter & aggregator

3. Launch

```
docker compose -f docker-compose.build.yaml build
docker compose -f docker-compose.aggregator.yaml up
```
