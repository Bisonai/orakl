# Orakl Network Contracts v0.2

## Prerequisities

Install Foundry by following description at https://book.getfoundry.sh/getting-started/installation or by executing the command below.

```shell
curl -L https://foundry.paradigm.xyz | bash

  Running `foundryup` by itself will install the latest (nightly) precompiled binaries: `forge`, `cast`, `anvil`, and `chisel`

Running `foundryup` by itself will install the latest (nightly) precompiled binaries: `forge`, `cast`, `anvil`, and `chisel`

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
forge script {ContractScriptName} --broadcast --rpc-url {RPC}
```
