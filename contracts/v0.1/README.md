# Orakl Network Contracts v0.1

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
Every migration directory should contain JSON migration files that contain migration definitions, and `migration.lock` which stores which migration has already been executed.

The names of migration files should be consistent.
We recommend to use the following script to generate a new migration file.

```shell
MIGRATION_NAME=
touch `date +%Y%m%d%H%M%S_${MIGRATION_NAME}.json`
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
yarn deploy:localhost:aggregator
```

### Hardhat library conflict issue

- Follow the link to avoid package conflict https://ethereum.stackexchange.com/questions/143246/conflict-peer-dependencies-nomicfoundation-hardhat-deploy-ethers
- `"@nomiclabs/hardhat-ethers": "npm:hardhat-deploy-ethers@^0.3.0-beta.13"`
- Without setting `0.3.0-beta.13` it requires to install `@nomiclabs/hardhat-ethers`. After setting this dependency, it fails to compile proper types.

### Script

1. Run `scripts/generate-aggregator-deployments.cjs` to creates migration, wallets, and bulk JSON files.
2. Migration files are saved in migration folder while wallets and bulk files are saved in `scripts/\*\*/tmp/` folder.
3. Execute the following command.

```shell
node ./scripts/admin-aggregator/generate-aggregator-deployments.cjs \
  --pairs '["usd-krw", "jpy-usd", "joy-usdc"]' \
  --chain baobab
```
