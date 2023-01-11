# ICN contracts

## Installation

```
yarn install
```

## Compilation

```
yarn compile
```

## Deployment

Deployment scripts are stored in [`deploy`](deploy) directory.

### Localhost

For local testing, it is best to both launch node and deploy with a single command.
The command below can be used for launching local test network for both off-chain `yarn` scripts and Docker as well.

```
npx hardhat node --hostname 0.0.0.0
```

### Baobab

```
npx hardhat deploy --network baobab
```
