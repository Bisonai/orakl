## Contracts

### VRF
For testing on local node:
1. run a local node
`npx hardhat node`
2. deploy prepayment, coordinator and config
`npx hardhat run src/v0.1/scripts/vrf/1-deploy-vrf-prepayment.mjs --network localhost`
3. start listener, worker, reporter
`cd core`
`yarn start:listener:vrf`
`yarn start:worker`
`yarn start:reporter`
4. request a random number
`npx hardhat run src/v0.1/scripts/vrf/2-request-vrf-prepayment.mjs --network localhost`
5. read the random number
`npx hardhat run src/v0.1/scripts/vrf/3-read-random-number-prepayment.mjs --network localhost`

#### Proving Keys

```
registerProvingKey
deregisterProvingKey
```

#### Transfer Ownership
```
requestSubscriptionOwnerTransfer
acceptSubscriptionOwnerTransfer
```

### Modify Subscription

```
createSubscription
cancelSubcription
```

#### Modify Consumers

```
addConsumer
removeConsumer
```
