#!/bin/bash

cd cli

yarn cli chain insert --name localhost
yarn cli service insert --name VRF
yarn cli service insert --name DATA_FEED
yarn cli service insert --name REQUEST_RESPONSE

cd ..
node update-rr-migration.js
node update-hardhat-network.js

cd contracts/v0.1
prepayment_output=$(yarn deploy:localhost:prepayment)
rr_output=$(yarn deploy:localhost:rr)
rr_address=$(echo $rr_output | awk -F'deployed at ' '{print $2}' | awk '{print $1}')

cd ../../cli
yarn cli listener insert --chain localhost --service REQUEST_RESPONSE --address $rr_address --eventName DataRequested
yarn cli reporter insert --chain localhost --service REQUEST_RESPONSE --address 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266 --privateKey 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 --oracleAddress $rr_address