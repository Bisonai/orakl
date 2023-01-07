# Off-chain Request-Response Scripts

Before running scripts in this folder, one must deploy `RequestResponseCoordinator` and `RequestResponseConsumerContract`.
To deploy the smart contracts, run `npx hardhat deploy --network localhost`.

## Make request

```
npx hardhat --network localhost run make-request.ts
```

## Read response

```
npx hardhat --network localhost run read-response.ts
```
