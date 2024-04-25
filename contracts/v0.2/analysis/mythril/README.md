# Mythril

```shell
pip3 install mythril
```

```shell
cd contracts/v0.2
myth analyze src/Feed.sol --solc-json  analysis/mythril/mythril.config.json > analysis/mythril/Feed.txt
myth analyze src/FeedProxy.sol --solc-json  analysis/mythril/mythril.config.json > analysis/mythril/FeedProxy.txt
myth analyze src/FeedRouter.sol --solc-json  analysis/mythril/mythril.config.json > analysis/mythril/FeedRouter.txt
myth analyze src/SubmissionProxy.sol --solc-json  analysis/mythril/mythril.config.json > analysis/mythril/SubmissionProxy.txt
```
