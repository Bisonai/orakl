# Off-chain Request-Response Scripts

Before running scripts in this folder, one must deploy `RequestResponseCoordinator` and `RequestResponseConsumerMock`.
To deploy the smart contracts, run `npx hardhat deploy --network localhost`.

## Make request

```
npx hardhat run request-data.ts --network localhost
```

## Read response

```
npx hardhat run read-data.ts --network localhost
```
