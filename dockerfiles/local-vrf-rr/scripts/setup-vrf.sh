#!/bin/bash

if [ -z "$ADDRESS" ]; then
  echo "ADDRESS is not set in the .core-cli-contracts.env file. Exiting."
  exit 1
fi

if [ -z "$PRIVATE_KEY" ]; then
  echo "PRIVATE_KEY is not set in the .core-cli-contracts.env file. Exiting."
  exit 1
fi

cd cli || exit

yarn cli chain insert --name "$CHAIN"
yarn cli service insert --name VRF

cd .. || exit
vrf_keys_path="vrf-keys.json"
sk=$(jq -r '.sk' "$vrf_keys_path")
pk=$(jq -r '.pk' "$vrf_keys_path")
pkX=$(jq -r '.pkX' "$vrf_keys_path")
pkY=$(jq -r '.pkY' "$vrf_keys_path")
keyHash=$(jq -r '.keyHash' "$vrf_keys_path")

echo $keyHash

cd cli || exit
yarn cli vrf insert --chain "$CHAIN" --sk "$sk" --pk "$pk" --pkX "$pkX" --pkY "$pkY" --keyHash "$keyHash"

cd .. || exit
node update-vrf-migration.js "$pkX" "$pkY" "$ADDRESS"
node update-hardhat-network.js "$PROVIDER_URL"

if [ "$CHAIN" == "localhost" ]; then
  cd contracts/v0.1 || exit
  yarn deploy:"$CHAIN":prepayment
  yarn deploy:"$CHAIN":vrf
  cd ../../cli || exit
else
  cd cli || exit
fi

ORACLE_ADDRESS=0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512
if [ "$CHAIN" == "baobab" ]; then
  ORACLE_ADDRESS=0xB55977Be3E014C41bc1E6c432488f310E3533B24
elif [ "$CHAIN" == "cypress" ]; then
  ORACLE_ADDRESS=0x3F247f70DC083A2907B8E76635986fd09AA80EFb
fi

yarn cli listener insert --chain "$CHAIN" --service VRF --address "$ORACLE_ADDRESS" --eventName RandomWordsRequested
yarn cli reporter insert --chain "$CHAIN" --service VRF --address "$ADDRESS" --privateKey "$PRIVATE_KEY" --oracleAddress "$ORACLE_ADDRESS"