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


if [ "$CHAIN" == "localhost" ]; then
  cd .. || exit
  vrf_keys_path="vrf-keys.json"
  sk=$(jq -r '.sk' "$vrf_keys_path")
  pk=$(jq -r '.pk' "$vrf_keys_path")
  pkX=$(jq -r '.pkX' "$vrf_keys_path")
  pkY=$(jq -r '.pkY' "$vrf_keys_path")
  keyHash=$(jq -r '.keyHash' "$vrf_keys_path")

  cd cli || exit
  yarn cli vrf insert --chain localhost --sk "$sk" --pk "$pk" --pkX "$pkX" --pkY "$pkY" --keyHash "$keyHash"

  cd .. || exit
  node update-vrf-migration.js "$pkX" "$pkY" "$ADDRESS"
  node update-hardhat-network.js "$PROVIDER_URL"
  cd contracts/v0.1 || exit
  yarn deploy:localhost:prepayment
  yarn deploy:localhost:vrf
  cd ../../cli || exit
else
  # set keyHash values for baobab/cypress chains:
  sk=""
  pk=""
  pkX=""
  pkY=""
  keyHash=""

  if [[ -z "$sk" || -z "$pk" || -z "$pkX" || -z "$pkY" || -z "$keyHash" ]]; then
    echo "VRF keyHash values are not set in the setup-vrf.sh file. Exiting."
    exit 1
  fi

  yarn cli vrf insert --chain "$CHAIN" --sk "$sk" --pk "$pk" --pkX "$pkX" --pkY "$pkY" --keyHash "$keyHash"
fi

ORACLE_ADDRESS=0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512
if [ "$CHAIN" == "baobab" ]; then
  ORACLE_ADDRESS=0xDA8c0A00A372503aa6EC80f9b29Cc97C454bE499
elif [ "$CHAIN" == "cypress" ]; then
  ORACLE_ADDRESS=0x3F247f70DC083A2907B8E76635986fd09AA80EFb
fi

yarn cli listener insert --chain "$CHAIN" --service VRF --address "$ORACLE_ADDRESS" --eventName RandomWordsRequested
yarn cli reporter insert --chain "$CHAIN" --service VRF --address "$ADDRESS" --privateKey "$PRIVATE_KEY" --oracleAddress "$ORACLE_ADDRESS"