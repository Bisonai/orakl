# Request-Response Scripts

Before running scripts in this folder, one must deploy `RequestResponseCoordinator` and `RequestResponseConsumerMock`.
To deploy the smart contracts, run `npx hardhat deploy --network localhost`.

## Request data

```shell
npx hardhat run request-data.cjs --network localhost
```

## Request data with direct payment

```shell
npx hardhatrun request-data-direct.cjs --network localhost
```

## Read data response

```shell
npx hardhat run read-data.cjs --network localhost
```
