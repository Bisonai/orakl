# Inspector

## Deploy contract

`npx hardhat deploy --network baobab`

## Run Script

### Create Account

- Creates orakl account

`npx hardhat run scripts/createAccount.ts --network baobab`

### Add Consumer

- Adds deployed InspectorConsumerContract to the account consumer
- `ACC_ID` which stands for account id should be set as environment variable

`npx hardhat run scripts/addConsumer.ts --network baobab`

### Fund Account

- Funds 5 klay to orakl account
- `ACC_ID` which stands for account id should be set as environment variable

`npx hardhat run scripts/fundAccount.ts --network baobab`

### requestAndRead

- Reads values before and after requesting rr & vrf
- Checks rr,vrf, and df for value changes (roundId for df)

`npx hardhat run scripts/requestAndRead.ts --network baobab`
