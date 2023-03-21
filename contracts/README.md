# Orakl Network Contracts

## Installation

```shell
yarn install
```

## Compilation

```shell
yarn compile
```

## Package

```shell
yarn build
```

## Deployment

Deployment scripts are stored in [`deploy` directory](deploy).
The deployment scripts are separate based on `service` (Aggregator, Prepayment, Request-Response, VRF).
Each script can handle deployment to any `network`, but the `network` has to be specified in `hardhat.config.ts` within `networks` object, and there must be a migration file for the specific `service` and `network`.
The general path for deployment script is `deploy/${network}/${service}`.

### Migration

Migration files are stored under [`migration` directory](migration).
The migration files are separated based on the `network` and `service`.
The general path to migration directory is `migration/${network}/${service}`.
Every migration directory should contain JSON migration files that contain migratino definitions, and `migration.lock` which stores which migration has already been executed.

The names of migration files should be consistent.
We recommend to use the following script to generate a new migration file.

```shell
MIGRATION_NAME=
touch `date +%Y%m%d%H%M%S_${MIGRATION_NAME}`
```

### Local Deployment
For local testing, it is best to both launch node and deploy with a single command.
The command below can be used for launching local test network.

```shell
npx hardhat node --hostname 127.0.0.1 --no-deploy
```

### VRF

```shell
yarn deploy:localhost:prepayment
yarn deploy:localhost:vrf
```

### Request-Response

```shell
yarn deploy:localhost:prepayment
yarn deploy:localhost:rr
```

### Aggregator

```shell
yarn deploy:aggregator
```
