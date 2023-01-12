# ICN core

## Prerequisites

Launch redis.

On MacOS install with `brew install redis` and launch with `brew services start redis`.

## Run

```
yarn install
```

```
yarn start:listener:vrf
yarn start:listener:icn
yarn start:worker
yarn start:reporter
```

## Docker

```
docker compose -f docker-compose.build.dev.yaml build
docker compose -f docker-compose.dev.yaml up
```

## Run Cli script

### How to run Price Feed script

Export adapterId param

```shell
export ADAPTERID=
```

Run

```shell
yarn clean
yarn build
yarn price_feed --adapterId ${ADAPTERID}
```
