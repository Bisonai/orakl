# Inspector

## Deploy `InspectorConsumer` Contract

```shell
npx hardhat deploy --network baobab
```

## Scripts

### Create Orakl Network Account

```shell
npx hardhat run scripts/createAccount.ts --network baobab
```

### Add Consumer

- Add deployed `InspectorConsumer` contract to the consumer account.
- Environment variable `ACC_ID` must be set before running the script below.

```shell
npx hardhat run scripts/addConsumer.ts --network baobab
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
