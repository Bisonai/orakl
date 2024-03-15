# Offchain Aggregator

## Introduction

Offchain aggregator.
It performs following steps to regularly submit data into chain

1. Fetch price data and save into database
2. Send and Recieve data with other nodes and aggregate all received data. save aggregated data into database
3. Submit aggregated data into chain

![Overview](./image.svg)

- `submitter` is `reporter`.
- Set of `fetcher`, `aggregator`, and `reporter` runs in a single application, assuming running in different instance.

## Project Structure

Modular Monolithic with loose coupling between packages

- cmd

Holds entrypoints to run basic functionalities

- migrations

Migration files to initialize pgsql table

- pkg

Implementation packages for offchain aggregator

- script

Scripts for testing purpose or temporary usage

- taskfiles

Taskfile holding commands to run application

## Packages

Check source code inside ./pkg for details

### Admin

Gofiber application for user interface. Mainly performs 2 things

1. CRUD for system tables
2. Control other package through bus message. (ex. stop fetcher)

### Aggregator

Aggregator shares recently fetched data with other aggregators, saves as global_aggregate.

### Bus

Base package for message bus communication among multiple packages.

### DB

Helper package to perform querying pgsql or redis

### Fetcher

Regularly fetches data from data source, saves into database

### Libp2p

Helper package to utilize libp2p package in a higher level

### Raft

Simple raft consensus implementation without log replication for leader election and syncing among multiple peers

### Reporter

Regularly report global_aggregates with latest Round into chain

### Utils

Includes helper functions to be used among other packages

## Main Elements

1. API

API supports interface to add entries to the table or control internal applications

2. Fetcher

Fetcher continuously fetches data from data source for entries declared in adapters table

3. Aggregator

Aggregator sends & recieves local fetched data into other offchain aggregators and saves into global_aggregates table

4. Reporter

Reporter submits all the data in global_aggregates with most recent round

## Quickstart

### Prerequisites

- go: https://go.dev/doc/install
- golang-migrate: https://github.com/golang-migrate/migrate/releases
- go-taskfile: https://taskfile.dev/installation/
- pgsql: https://www.postgresql.org/download/
- redis: https://redis.io/docs/install/install-redis/install-redis-on-linux/

### Setup DB and ENV

```bash
APP_PORT=${app port for pai} #defaults to 3000
DATABASE_URL=${postgresql connection url}
REDIS_HOST=${redis_host} #defaults to localhost
REDIS_PORT=${redis_port} #defaults to 6379
ENCRYPT_PASSWORD=${encrypt password for topic string encryption.}
BOOT_NODE=${libp2p addr for bootnode, not required}
LISTEN_PORT=${libp2p listen port}
PROVIDER_URL=${target chain for submission}
CONFIG_URL=${orakl config url for adapter json file}
REPORTER_PK=${reporter for submission, not required if entry is inside wallets table}
SUBMISSION_PROXY_CONTRACT=${contract for submission}
DELEGATOR_URL=${delegator url, not required}
TEST_FEE_PAYER_PK=${referenced from testcode, eoa of fee payer}
```

### Run unit tests

- Run all tests

```
task local:test
```

> check out `./taskfiles/taskfile.local/yml` to check command for certain test

### Run API

```
task local:admin
```

### Run Scripts

- Temporary scripts used to run application in testing environment, to be removed after stablized. Use it to test application in local environment

- Submission test: submit single tx on chain

```
task local:script-submission
```

- Fetcher test: run api + fetcher

```
task local:script-fetcher-test
```

- Fetcher-aggregator test: run api + fetcher + aggregator

```
task local:script-fetcher-aggregator-test
```

- All: run api + fetcher + aggregator + reporter

```
task local:script-test-all
```
