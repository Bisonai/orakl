# Inspector

## Deploy Consumer Contracts

```shell
# deploy the InspectConsumer contract
npx hardhat deploy --network baobab

# deploy the LoadTestVRFConsumer contract
npx hardhat deploy --network baobab --tags load-test-vrf-consumer

# deploy the LoadTestRRConsumer contract
npx hardhat deploy --network baobab --tags load-test-rr-consumer
```

## Scripts

### Create Orakl Network Account

```shell
npx hardhat run scripts/createAccount.ts --network baobab
```

### Add Consumer

- Add deployed `InspectorConsumer`, `LoadTestVRFConsumer`, and `LoadTestRRConsumer` contracts to the consumer account.
- Environment variable `ACC_ID` must be set before running the script below.

```shell
# add InspectorConsumer
npx hardhat addConsumer --network baobab --consumer inspector

# add LoadTestVRFConsumer
npx hardhat addConsumer --network baobab --consumer vrf

# add LoadTestRRConsumer
npx hardhat addConsumer --network baobab --consumer rr
```

### Fund Account

- Fund 5 $KLAY to Orakl Network account.
- Environment variable `ACC_ID` must be set before running the script below.

```shell
npx hardhat run scripts/fundAccount.ts --network baobab
```

### Request And Read

1. Read the last fulfilled values for VRF & RR.
2. Create a new requests for VRF & RR.
3. After a short while, read the last fulfilled values again.
4. Compare old and new fulfilled values, and see whether they changed.
5. If there are no changes in values, fulfillment has not been performed or there is a delay in the system.

- Environment variable `ACC_ID` must be set before running the script below.

```shell
npx hardhat run scripts/requestAndRead.ts --network baobab
```

### Request And Read (Hardhat Task)

```shell
# baobab vrf
npx hardhat inspect --network baobab --service vrf
# baobab rr
npx hardhat inspect --network baobab --service rr
# baobab all
npx hardhat inspect --network baobab

# cypress vrf
npx hardhat inspect --network cypress --service vrf
# cypress rr
npx hardhat inspect --network cypress --service rr
# cypress all
npx hardhat inspect --network cypress
```

### Load Test VRF and RR

```shell
# VRF
npx hardhat load-test-vrf --network baobab --batch n

# RR
npx hardhat load-test-rr --network baobab --batch n
```

Replace `n` with any number of batches you'd like to run. Each batch will make 50 requests. Every batch is awaited to be mined before making the next batch request. The results of each request are measured by the number of blocks it takes to fulfill, which is equivalent to seconds. The consumer contract keeps track of each request's requestId and computes the time it takes to fulfill the request.
