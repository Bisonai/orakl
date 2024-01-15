# Insepctor

## Deploy contract

`npx hardhat deploy --network baobab`

## Run Script

### requestAndRead

- Reads values before and after requesting rr & vrf
- Checks rr,vrf, and df for value changes (roundId for df)

`npx hardhat run scripts/requestAndRead.ts --network baobab`
