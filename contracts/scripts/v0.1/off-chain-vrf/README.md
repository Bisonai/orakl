# Off-chain VRF Scripts

Before running scripts in this folder, one must deploy `VRFCoordinator` and `VRFConsumerContract`.
To deploy the smart contracts, run `npx hardhat deploy --network localhost`.

## Request VRF

```
npx hardhat --network localhost run request-vrf.mjs
```

## Read VRF response

```
npx hardhat --network localhost run read-vrf.mjs
```
