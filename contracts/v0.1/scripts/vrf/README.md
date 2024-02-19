# VRF Scripts

Before running scripts in this folder, one must deploy `VRFCoordinator` and `VRFConsumerMock`.
To deploy the smart contracts, run `npx hardhat deploy --network localhost`.

## Request VRF

```shell
npx hardhat run request-vrf.cjs --network localhost
```

## Request VRF with direct payment

```shell
npx hardhat run request-vrf-direct.cjs --network localhost
```

## Read VRF response

```shell
npx hardhat run read-vrf.cjs --network localhost
```
