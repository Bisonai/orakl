# Data Feed Scripts

Before running scripts in this folder, one must deploy `Aggregator`, `AggregatorProxy` and `DataFeedConsumerMock`.

To deploy the smart contracts, run `npx hardhat deploy --network localhost`.

## Read oracle round state

```shell
npx hardhat run oracle-round-state.cjs --network localhost
```
