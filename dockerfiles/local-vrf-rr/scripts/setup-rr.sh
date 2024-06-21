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
yarn cli service insert --name REQUEST_RESPONSE

cd .. || exit
node update-rr-migration.js "$ADDRESS"
node update-hardhat-network.js "$PROVIDER_URL"

if [ "$CHAIN" == "localhost" ]; then
  cd contracts/v0.1 || exit
  yarn deploy:"$CHAIN":prepayment
  yarn deploy:"$CHAIN":rr
  cd ../../cli || exit
else
  cd cli || exit
fi

ORACLE_ADDRESS=0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512
if [ "$CHAIN" == "baobab" ]; then
  ORACLE_ADDRESS=0x0d2fAcdd858F04AAb1263Ab10f3a04A512E0640C
elif [ "$CHAIN" == "cypress" ]; then
  ORACLE_ADDRESS=0x159F3BB6609B4C709F15823F3544032942106042
fi

yarn cli listener insert --chain "$CHAIN" --service REQUEST_RESPONSE --address "$ORACLE_ADDRESS" --eventName DataRequested
yarn cli reporter insert --chain "$CHAIN" --service REQUEST_RESPONSE --address "$ADDRESS" --privateKey "$PRIVATE_KEY" --oracleAddress "$ORACLE_ADDRESS"