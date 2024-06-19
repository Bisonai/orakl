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
  - [Optional Setups](#optional-setups)
  - [Run Node](#run-node)
- [Other Task Commands](#other-task-commands)
  - [Unit Test](#unit-test)
  - [Commands](#commands)
  - [Scripts](#scripts)
  - [CLI](#cli)
- [Troubleshooting](#troubleshooting)
  - [`Klaytn` package compile error](#klaytn-package-compile-error)

## Introduction

Off-chain aggregator performs the following steps to regularly submit data into the chain:

1. Fetch price data and save it into the database.
2. Send and receive data with other nodes, aggregate all received data, and save aggregated data into the database.
3. Submit aggregated data into the chain.

<figure><img src="./Node.drawio.svg" alt=""><figcaption><p>Set of `Admin`, `fetcher`, `aggregator`, and `reporter` runs in a single Orakl Node</p></figcaption></figure>

<figure><img src="./DAL.drawio.svg" alt=""><figcaption><p>Data Availability Layer for both pull & push pattern</p></figcaption></figure>

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
- **Por**: Package to run a separate service for POR.
- **wss**: Helper package for websocket implementation
- **websocketfetcher**: Fetcher app based on websocket

### Main Elements

- **Admin API**: Supports an interface to add entries to the table or control internal applications.
- **Fetcher**: Continuously retrieves data from the data source for entries declared in the adapters table.
- **Aggregator**: Sends and receives locally fetched data to/from other off-chain aggregators, storing it in the `global_aggregates` table.
- **Reporter**: Submits all data from `global_aggregates` with the most recent round.

## Quickstart

### Prerequisites

Ensure you have the following installed and set up:

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

# provider URL for `kaia_helper`
KAIA_PROVIDER_URL=<Your Provider URL>

# provider URL for `eth_helper`
ETH_PROVIDER_URL=<Provider URL>

# Contract for submission
SUBMISSION_PROXY_CONTRACT=<Your Submission Proxy Contract>

# Delegator URL, tx fee is directly paid from reporter if not provided
DELEGATOR_URL=<Your Delegator URL>

# Signer PK generates a signature value for submission based on this value. EOA address should be whitelisted in the SubmissionProxy contract to be used.
SIGNER_PK=<Your Signer PK>

# Encrypt Password, this is referenced to store encrypted wallet pk into table. defaults to 'anything'
ENCRYPT_PASSWORD=<Your Encrypt Password>

# Chain name, 'baobab', 'cypress', or 'test'
CHAIN=<Your Chain Name>

# tx submission wallet for `kaia_helper`
KAIA_REPORTER_PK=<Your Reporter PK>

# tx submission wallet for `eth_helper`
ETH_REPORTER_PK=<Reporter PK>

# Referenced from test code, EOA of fee payer
TEST_FEE_PAYER_PK=<Your Test Fee Payer PK>

# Required for secure connection
PRIVATE_NETWORK_SECRET=<Your Private Network Secret>

# Port for Boot API, defaults to 8089
BOOT_API_PORT=<Your Boot API Port>

# Boot API connection URL
BOOT_API_URL=<Your Boot API URL>

# provider URLs referenced from fetcher, uses public JSON-RPC if not provided
FETCHER_CYPRESS_PROVIDER_URL=<Your Cypress provider URL>
FETCHER_ETHEREUM_PROVIDER_URL=<Your Ethereum provider URL>
```

### Database Initialization

After go-migrate is installed, run migration with the following command:
More details about go migrate cli command can be found [here](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate)

```sh
# node
migrate -database "{$DATABASE_URL}" -path ./migrations/node up

# boot
migrate -database "{$DATABASE_URL}" -path ./migrations/boot up
```

This process will generate required tables and constraints to run the service.

### Optional Setups

Following are optional setups which helps the application run more smoothly.

1. Proxies

Proxies are referenced from fetcher to prevent being blocked from 3rd party data providers. If provided, fetcher will utilize proxy requests

2. Wallets

Wallets are referenced from reporter. If provided, each wallet will take a turn for submissions.

3. JSON-RPCs

JSON-RPCs are referenced from both fetcher and reporter. If provided, it'll try to use provided JSON-RPCs as a fallback in case of JSON-RPC failure.

---

If you want to set these settings, use [cli commands](#cli) while admin API is running. Admin API is run together while the node is running, or you can run Admin API separately without running the whole service through the following task command

```sh
task local:admin
```

- Run CLI commands while the admin API is running (e.g., `task local:add-wallet PK=0x123`).
- If the whole service was running when adding the settings, refresh the related service to apply changes (e.g., `task local:refresh-reporter`).

### Run Node

Follow these steps to set up and run the application:

1. **Set up the database**: Ensure PostgreSQL and Redis are running. PostgreSQL should have tables based on migration files.
2. **Copy .env.local to .env**: Copy the local environment settings to the main environment file

```sh
cp .env.local .env
```

3. **Update environment variables**: Replace `DATABASE_URL`, `KAIA_REPORTER_PK`, `SIGNER_PK`, and other values with valid ones in the .env file.
4. **Run Boot API**:
   skip this step if connecting to pre-existing boot api

   ```sh
   task local:boot-api
   ```

5. **Run Node**: Execute this from a different shell.

   ```sh
   task local:node
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

# Run Node
task local:node
```

### Scripts

```sh
# Submission test: submit a single tx on chain
task local:script-submission

# Fetcher test: run api + fetcher
task local:script-fetcher-test

# Fetcher-aggregator test: run api + fetcher + aggregator
task local:script-fetcher-aggregator-test

# Test connection: check if nodes properly connect through boot api
task local:script-test-connection

# Test raft: run simple raft node to test its functionality
task local:script-test-raft
```

### CLI

```sh
# check if admin api is live
task local:check-api

# refresh fetcher, trigger it after adding proxy or json rpc. Execute only if service is running.
task local:refresh-fetcher

# refresh reporter, trigger it after adding wallet or json rpc. Execute only if service is running.
task local:refresh-reporter

# refresh aggregator, reloads aggregators and signers.
task local:refresh-aggregator

# `LOCATION` is an optional parameter
task local:add-proxy HOST="127.0.0.2" PORT=8080 PROTOCOL="http" LOCATION="kr"

# get all registered proxies
task local:get-proxy

# remove proxy by id
task local:remove-proxy ID=10

# add a wallet which triggers submission
task local:add-wallet PK=0x123

# get all registered wallets
task local:get-wallet

# remove wallet by id
task local:remove-wallet ID=10

# add fallback JSON-RPC, lower priority value will be referenced first
task local:add-json-rpc CHAIN_ID=1001 URL="http://test.com" PRIORITY=10

# get all registered fallback JSON-RPC
task local:get-json-rpc

# remove fallback json-rpc
task local:remove-json-rpc ID=10
```

## Troubleshooting

### `Klaytn` package compile error

[Reference](https://github.com/klaytn/klaytn/issues/197#issuecomment-612597933)

1. Install C compilers

```sh
# use the appropriate command depending on the instance environment
sudo apt-get install -y g++-x86-64-linux-gnu libc6-dev-amd64-cross
```

2. Set variables

```sh
set CGO_ENABLED=1
set CC=[c cross compiler]
set GOOS=linux
set GOARCH=amd64
```

# POR

Por service stands for proof of reserve. To be updated later, with scalability to support multiple providers.

## Quickstart

### Set .env variables

```sh
# POR
POR_REPORTER_PK=
POR_CHAIN=
POR_PROVIDER_URL=
# (optional) defaults to 3000
POR_PORT=
```

### Run

```sh
task local:por
```
