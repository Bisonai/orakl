# Orakl Network Contracts v0.2

## Prerequisities

Install Foundry by following description at https://book.getfoundry.sh/getting-started/installation or by executing the command below.

```shell
curl -L https://foundry.paradigm.xyz | bash
```

## Build

```shell
forge build
```

## Test

```shell
forge test
```

## Format

```shell
forge fmt
```

## Deployment

1. Create `.env` from `.env.example` and fill in `PRIVATE_KEY`

```
cp .env.example .env
```

2. Deploy

```shell
forge script deploy/SubmissionProxy.s.sol:SubmissionProxyScript --rpc-url [RPC] --broadcast
forge script deploy/Aggregator.s.sol:AggregatorScript --rpc-url [RPC] --broadcast
```
