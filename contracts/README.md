# ICN contracts

## Compilation

```
yarn compile
```

# Run Hardhat Local Node

```
npx hardhat node
```

# Testing Oracle Requests


1. Run a local Hardhat Network

```
npx hardhat node
```

2. Run the event listening script replacing private key with PK returned from local hardhat node

```
npx hardhat run src/v0.1/scripts/eventListener.mjs --network localhost
```
