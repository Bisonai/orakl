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

## Local Bull Queue Monitoring

```
docker compose -f docker-compose.bull-monitor.yaml up
```

Bull Queue Board: http://localhost:3000/queues/