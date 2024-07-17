# Orakl Network

This repository is split to [on-chain](contracts) and off-chain oracle implementation.

You can learn more about the Orakl Network from [documentation](https://orakl-network.gitbook.io).

<details>
<summary> Run dockers locally </summary>

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

</details>

## Main Elements

### API

The API serves as an interface for other services (VRF, RR, CLI) to access service data.

### CLI

Includes commands that node operators can use to manage VRF and RR.

### Contracts

Contains all on-chain contracts for the service and related scripts.

### Core

Contains off-chain code for VRF and RR.

### Delegator

A service to run fee payer for delegated transactions by push oracles.

### Node

Contains off-chain code for off-chain aggregation and related services.

### Others

- Audit: Holds contract audit details.
- Dockerfiles: Includes Dockerfiles for deployment.
- Sentinel: Internal alarm service for stable service maintenance.

## Image tags

[image tag release](./TAGS.md)
