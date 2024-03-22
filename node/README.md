# Orakl Node

**Orakl Node for Off-Chain Aggregation**

## Table of Contents

- [Introduction](#introduction)
- [Project Structure](#project-structure)
- [Packages](#packages)
  - [Main Elements](#main-elements)
- [Quickstart](#quickstart)
  - [Prerequisites](#prerequisites)
  - [Setup `.env`](#setup-env)
  - [Database Initialization](#database-initialization)
  - [Test Run Single Local Node](#test-run-single-local-node)
- [Other Task Commands](#other-task-commands)
  - [Unit Test](#unit-test)
  - [Commands](#commands)
  - [Scripts](#scripts)

## Introduction

Off-chain aggregator performs the following steps to regularly submit data into the chain:

1. Fetch price data and save it into the database.
2. Send and receive data with other nodes, aggregate all received data, and save aggregated data into the database.
3. Submit aggregated data into the chain.

![Overview](./Node.drawio.svg)

- Set of `Admin`, `fetcher`, `aggregator`, and `reporter` runs in a single Orakl Node

![DAL](./DAL.drawio.svg)

- Data Availability Layer for both pull & push pattern

## Project Structure

Modular Monolithic with loose coupling between packages:

- `cmd`: Holds entry points to run basic functionalities.
- `migrations`: Migration files to initialize PostgreSQL tables.
- `pkg`: Implementation packages for off-chain aggregator.
- `script`: Scripts for testing purposes, such as smoke tests or temporary usage.
- `taskfiles`: Taskfile holding commands to run the application.

## Packages

Check the source code inside `./pkg` for details:

- **Boot**: Handles peer initial connection. One Boot API should be running for node meshes.
- **Admin**: Provides an API for the user interface, performing CRUD operations for system tables and controlling other packages through bus messages (e.g., stopping fetcher).
- **Aggregator**: Shares recently fetched data with other aggregators and saves it as `global_aggregate`.
- **Bus**: Facilitates message bus communication among multiple packages.
- **DB**: Offers helper functions for querying PostgreSQL or Redis databases.
- **Fetcher**: Regularly retrieves data from the data source and stores it in the database.
- **Libp2p**: Assists in utilizing the libp2p package at a higher level.
- **Raft**: Implements simple raft consensus for leader election and syncing among multiple peers.
- **Reporter**: Submits data from `global_aggregates` with the latest Round to the chain.
- **Utils**: Contains helper functions usable among other packages.

### Main Elements

- **Admin API**: Supports an interface to add entries to the table or control internal applications.
- **Fetcher**: Continuously retrieves data from the data source for entries declared in the adapters table.
- **Aggregator**: Sends and receives locally fetched data to/from other off-chain aggregators, storing it in the `global_aggregates` table.
- **Reporter**: Submits all data from `global_aggregates` with the most recent round.

## Quickstart

### Prerequisites

Ensure you have the following installed:

- Go: [Installation Guide](https://go.dev/doc/install)
- golang-migrate: [Installation Guide](https://github.com/golang-migrate/migrate/releases)
- go-taskfile: [Installation Guide](https://taskfile.dev/installation/)
- PostgreSQL: [Installation Guide](https://www.postgresql.org/download/)
- Redis: [Installation Guide](https://redis.io/docs/install/install-redis/install-redis-on-linux/)

### Setup `.env`

```sh
# Application port for the admin API, defaults to 8088
APP_PORT=<Your App Port>

# PostgreSQL connection URL
DATABASE_URL=<Your Database URL>

# Redis host, defaults to localhost
REDIS_HOST=<Your Redis Host>

# Redis port, defaults to 6379
REDIS_PORT=<Your Redis Port>

# libp2p listen port
LISTEN_PORT=<Your Listen Port>

# Chain provider URL
PROVIDER_URL=<Your Provider URL>

# Contract for submission
SUBMISSION_PROXY_CONTRACT=<Your Submission Proxy Contract>

# Delegator URL, tx fee is directly payed from reporter if not provided
DELEGATOR_URL=<Your Delegator URL>

# Chain name, 'baobab', 'cypress', or 'test'
CHAIN=<Your Chain Name>

# Reporter for submission, not required if entry is inside wallets table
REPORTER_PK=<Your Reporter PK>

# Referenced from test code, EOA of fee payer
TEST_FEE_PAYER_PK=<Your Test Fee Payer PK>

# Required for secure connection
PRIVATE_NETWORK_SECRET=<Your Private Network Secret>

# Port for Boot API, defaults to 8089
BOOT_API_PORT=<Your Boot API Port>

# Boot API connection URL
BOOT_API_URL=<Your Boot API URL>
```

### Database Initialization

After go-migrate is installed, run migration with the following command:

```sh
migrate -database "{$DATABASE_URL}" -path ./migrations up
```

### Test Run Single Local Node

Follow these steps to set up and run the application:

1. **Set up the database**: Ensure PostgreSQL and Redis are running. PostgreSQL should have tables based on migration files.
2. **Copy .env.local to .env**:

    ```sh
    cp .env.local .env
    ```

3. **Update environment variables**: Replace `DATABASE_URL` and `REPORTER_PK` with valid values in the .env file.
4. **Run Boot API**:

    ```sh
    task local:boot-api
    ```

5. **Run test-all script**: Execute this from a different shell.

    ```sh
    task local:script-test-all
    ```

## Other Task Commands

### Unit Test

```sh
# Run all tests
task local:test
```

### Commands

```sh
# Run Boot API
task local:boot-api

# Run Admin API
task local:admin
```

### Scripts

```sh
# Submission test: submit single tx on chain
task local:script-submission

# Fetcher test: run api + fetcher
task local:script-fetcher-test

# Fetcher-aggregator test: run api + fetcher + aggregator
task local:script-fetcher-aggregator-test

# All: run api + fetcher + aggregator + reporter
task local:script-test-all

# Test connection: check if nodes properly connect through boot api
task local:script-test-connection

# Test raft: run simple raft node to test its functionality
task local:script-test-raft
```

## Troubleshooting

### `Klaytn` package compile error

[Reference](https://github.com/klaytn/klaytn/issues/197#issuecomment-612597933)

1. Install C compilers
```sh
# use appropriate command depending on instance environment
sudo apt-get install -y g++-x86-64-linux-gnu libc6-dev-amd64-cross
```

2. Set variables
```sh
set CGO_ENABLED=1
set CC=[c cross compiler]
set GOOS=linux
set GOARCH=amd64
```