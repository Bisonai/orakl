#!/bin/bash

cd cli || exit

yarn cli chain insert --name localhost
yarn cli service insert --name VRF
yarn cli service insert --name REQUEST_RESPONSE

keygen_output=$(yarn cli vrf keygen)
sk=$(echo "$keygen_output" | awk -F'sk=' '{print $2}' | awk '{print $1}')
pk=$(echo "$keygen_output" | awk -F'pk=' '{print $2}' | awk '{print $1}')
pkX=$(echo "$keygen_output" | awk -F'pkX=' '{print $2}' | awk '{print $1}')
pkY=$(echo "$keygen_output" | awk -F'pkY=' '{print $2}' | awk '{print $1}')
keyHash=$(echo "$keygen_output" | awk -F'keyHash=' '{print $2}' | awk '{print $1}')

yarn cli vrf insert --chain localhost --sk "$sk" --pk "$pk" --pkX "$pkX" --pkY "$pkY" --keyHash "$keyHash"

cd .. || exit
node update-vrf-migration.js $pkX $pkY
node update-hardhat-network.js

cd contracts/v0.1 || exit
yarn deploy:localhost:prepayment
yarn deploy:localhost:vrf

cd ../../cli || exit
yarn cli listener insert --chain localhost --service VRF --address 0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512 --eventName RandomWordsRequested
yarn cli reporter insert --chain localhost --service VRF --address 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266 --privateKey 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 --oracleAddress 0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512

tail -f /dev/null