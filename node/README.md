# Off-chain Aggregator

## Introduction

Offchain aggregator.
It performs the following steps to regularly submit data into chain

1. Fetch price data and save into database
2. Send and Receive data with other nodes and aggregate all received data. Save aggregated data into database
3. Submit aggregated data into chain

![Overview](./Node.drawio.svg)

- Set of `fetcher`, `aggregator`, and `reporter` runs in a single application, running in different instance.

![Dal](./DAL.drawio.svg)

- Data Availability Layer for both pull & push pattern orakl implementation.

## Project Structure

Modular Monolithic with loose coupling between packages

- cmd

Holds entry points to run basic functionalities

- migrations

Migration files to initialize PostgreSQL table

- pkg

Implementation packages for offchain aggregator

- script

Scripts for testing purpose or temporary usage

- taskfiles

Taskfile holding commands to run application

## Packages

Check source code inside ./pkg for details

### Boot

Boot Api for peer initial connection. One boot api should be running for node meshes.

### Admin

Gofiber application for user interface. Mainly performs 2 things

1. CRUD for system tables
2. Control other packages through bus messages. (For example, stop fetcher)

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

Aggregator sends & receives local fetched data into other off-chain aggregators and saves into global_aggregates table

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

```sh
APP_PORT=${app port for pai} #defaults to 3000
DATABASE_URL=${postgresql connection url}
REDIS_HOST=${redis_host} #defaults to localhost
REDIS_PORT=${redis_port} #defaults to 6379
LISTEN_PORT=${libp2p listen port} # should allow inbound connection for this tcp port
PROVIDER_URL=${chain provider url}
SUBMISSION_PROXY_CONTRACT=${contract for submission}
DELEGATOR_URL=${delegator url, not required}
CHAIN=${chain name, `baobab` or `cypress`}
REPORTER_PK=${reporter for submission, not required if entry is inside wallets table}
TEST_FEE_PAYER_PK=${referenced from testcode, eoa of fee payer}
PRIVATE_NETWORK_SECRET=${required for secure connection}

# required to run boot api
BOOT_API_PORT=${defaults to 8089}

# required for node connection
BOOT_API_URL=${boot api connection url}
```

### Database initialization

- If `go-migrate` is installed, run migration with following command. Using `?sslmode=disable` option for database url is recommended.

```sh
migrate -database "{$DATABASE_URL}" -path ./migrations up
```

### Run unit tests

- Run all tests

```sh
task local:test
```

> check out `./taskfiles/taskfile.local.yml` to check command for certain test

### Run Boot API

```sh
task local:boot-api
```

### Run API

```sh
task local:admin
```

### Run Scripts

- Submission test: submit single tx on chain

```sh
task local:script-submission
```

- Fetcher test: run api + fetcher

```sh
task local:script-fetcher-test
```

- Fetcher-aggregator test: run api + fetcher + aggregator

```sh
task local:script-fetcher-aggregator-test
```

- All: run api + fetcher + aggregator + reporter

```sh
task local:script-test-all
```

- test connection: check if nodes properly connects through boot api

```sh
task local:script-test-connection
```

- test raft: run simple raft node to test its functionality

```sh
task local:script-test-raft
```
