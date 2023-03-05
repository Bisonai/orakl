# Orakl Network Core

## Local development

On MacOS install with `brew install redis` and launch with `brew services start redis`.

## Build & Run

```
yarn install
```

### VRF

```
yarn start:listener:vrf
yarn start:worker:vrf
yarn start:reporter:vrf
```

### Aggregator

```
yarn start:listener:aggregator
yarn start:worker:aggreagator
yarn start:reporter:aggregator
```

## Production

```
docker compose -f docker-compose.build.yaml build

# Launch VRF
docker compose -f docker-compose.vrf.yaml up

# Launch Aggregator
docker compose -f docker-compose.aggregator.yaml up
```

## Local Bull Queue Monitoring

```
docker compose -f docker-compose.bull-monitor.yaml up
```

Bull Queue Board: http://localhost:3001/queues

## Run cli script

### Run price-feed

```shell
export ADAPTERID=
```

Run

```shell
yarn price_feed --adapterId ${ADAPTERID}
```
