# Off-chain Data Feed Scripts

Before running scripts in this folder, one must deploy `Aggregator`, `AggregatorProxy` and `DataFeedConsumerMock`.

To deploy the smart contracts, run `npx hardhat deploy --network localhost`.

## Read data

```
npx hardhat --network localhost run read-data.mjs
```
