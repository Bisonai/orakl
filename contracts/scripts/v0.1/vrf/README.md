# VRF Scripts

Before running scripts in this folder, one must deploy `VRFCoordinator` and `VRFConsumerMock`.
To deploy the smart contracts, run `npx hardhat deploy --network localhost`.

## Request VRF

```
npx hardhat run request-vrf.ts --network localhost
```

## Request VRF with direct payment

```
npx hardhat run request-vrf-direct.ts --network localhost
```

## Read VRF response

```
npx hardhat run read-vrf.ts --network localhost
```
