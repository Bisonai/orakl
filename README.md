# Orakl Network

This repository is split to [on-chain](contracts) and [off-chain](core) oracle implementation.

You can learn more about the Orakl Network from [documentation](https://orakl-network.gitbook.io).

# Test Data Feed Locally Using Docker

Run local data feed connected to testnet.

## Steps run through docker compose

- Deploy contracts in testnet(baobab)
- Run postgres & redis
- Run orakl-api & orakl-delegator
- Insert deployed data feed & set delegator fee payer
- Run listener, worker, reporter, and fetcher
- Activate inserted data feed

## Prerequisites

1. Docker

```bash
brew install docker
brew install docker-compose
```

2. Env setup

Nearly everything is already setup, but there are two variables that should be set manually in following env files:

- `dockerfiles/local-data-feed/envs/.contracts.env`

```
MNEMONIC="{MNEMONIC for contract deployer wallet}"
```

- `dockerfiles/local-data-feed/envs/.cli.env`

```
DELEGATOR_REPORTER_PK={private key for delegator fee payer}
```

## Run docker

### Data feed

1. Docker Compose Build
   Builds all required images for docker-compose.

```bash
docker-compose -f docker-compose.local-data-feed.yaml build
```

2. Docker Compose Up
   Runs all required images to run datafeed locally.

```bash
docker-compose -f docker-compose.local-data-feed.yaml up
```

3. Docker Compose Down
   Close all related containers.

```bash
docker-compose -f docker-compose.local-data-feed.yaml down -v
```

### VRF / Request-Response

Run the following command to build all images

```bash
docker-compose -f docker-compose.local-core.yaml build
```

Set wallet credentials, `ADDRESS` and `PRIVATE_KEY` values, in the [.core-cli-contracts.env](./dockerfiles/local-vrf-rr/envs/.core-cli-contracts.env) file. You can update `CHAIN` to either `baobab` and `cypress`. For running the `VRF` service, update [vrf-keys-baobab.json](./dockerfiles/local-vrf-rr/envs/vrf-keys-baobab.json) or [vrf-keys-cypress.json](./dockerfiles/local-vrf-rr/envs/vrf-keys-cypress.json). If the chain is not `localhost`, `Coordinator` and `Prepayment` contracts won't be deployed. Instead, Bisonai's already deployed contract addresses ([VRF](https://github.com/Bisonai/vrf-consumer/blob/master/hardhat.config.ts), [RR](https://github.com/Bisonai/request-response-consumer/blob/master/hardhat.config.ts)) will be used. Run the following command to start the VRF / RequestResponse service:

```bash
SERVICE=vrf docker-compose -f docker-compose.local-core.yaml up --force-recreate -d
```

`SERVICE` is an env variable that will be used to spin up the specified service. The options are `rr` and `vrf` stands for Request-Response and VRF, respectively.

**Note** that the current docker implementation is designed to run a single service, either `rr` or `vrf`, at a time. Therefore, it's highly recommended to add `--force-recreate` when running `docker-compose up` command. That will restart all containers thus removing all the modified data in those containers.

Here is what happens after the above command is run:

- `api`, `postgres`, `redis`, and `json-rpc` services will start as separate docker containers
- `postgres` will get populated with necessary data:
  - chains
  - services
  - vrf keys in case if the service is vrf
  - listener (after contracts are deployed)
  - reporter (after contracts are deployed)
- if the chain is `localhost`:
  - migration files in `contracts/v0.1/migration/` get updated with provided keys and other values
  - `contracts/v0.1/hardhat.config.cjs` file gets updated with `PROVIDER_URL`
  - relevant coordinator and prepayment contracts get deployed

To start core microservices (listener, worker, reporter) as a singleton service run:

- production mode
  ```sh
  yarn start:core:vrf
  # or
  yarn start:core:request_response
  ```
- development mode
  ```sh
  yarn dev:core:vrf
  # or
  yarn dev:core:request_response
  ```

The microservices can also be started separately in any arbitrary order by replacing `core` in the above commands with either `listener`, `worker`, or `reporter`

## Image tag release (do not write under this section)

- **node** v0.0.1.20240624.0427.15e4017 <br> _`PR`_: OraklNode Websocket dex fetche... <br><br>
- **node** v0.0.1.20240624.0527.8a67f20 <br> _`PR`_: OraklNode Update fetcher init <br><br>
- **sentinel** v0.0.1.20240624.0545.3e49d01 <br> _`PR`_: OraklNode Update fetcher init <br><br>
- **sentinel** v0.0.1.20240624.0757.8096587 <br> _`PR`_: send service is up slack messa... <br><br>
- **node** v0.0.1.20240624.0815.2b2ed32 <br> _`PR`_: OraklNode Update proxy request <br><br>
- **node** v0.0.1.20240624.0824.a6ca4ab <br> _`PR`_: OraklNode Index out of bound p... <br><br>
- **node** v0.0.1.20240624.2328.3517cf5 <br> _`PR`_: OraklNode Execute local aggreg... <br><br>
- **node** v0.0.1.20240624.2359.c658e51 <br> _`PR`_: OraklNode Reduce intervals <br><br>
- **sentinel** v0.0.1.20240625.0255.5673f9d <br> _`PR`_: Sentinel Add external signers ... <br><br>
- **node** v0.0.1.20240626.0736.d9a8df4 <br> _`PR`_: OraklNode Rollback klayswap ba... <br><br>
- **node** v0.0.1.20240626.0926.c3f91cb <br> _`PR`_: OraklNode Update sync <br><br>
- **node** v0.0.1.20240627.0431.bac3940 <br> _`PR`_: OraklNode Update klayswap to u... <br><br>
- **sentinel** v0.0.1.20240628.0621.ac5b604 <br> *`PR`*: Update dispatch deploy workflo... <br><br> 
- **sentinel** v0.0.1.20240628.0920.7fb4ffa <br> *`PR`*: Sentinel Update request packag... <br><br> 
- **node** v0.0.1.20240701.0129.82bd4e0 <br> *`PR`*: OraklNode Remove unused column... <br><br> 
- **node** v0.0.1.20240701.0528.b6b6a7e <br> *`PR`*: OraklNode Remove unused column... <br><br> 
- **node** v0.0.1.20240701.0602.97ca5e2 <br> *`PR`*: OraklNode PubSub based GlobalA... <br><br> 
- **node** v0.0.1.20240701.0650.4eaa326 <br> *`PR`*: OraklNode Reduce default lates... <br><br> 
