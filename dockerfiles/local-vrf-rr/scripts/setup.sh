#!/bin/bash

cd cli

yarn cli chain insert --name localhost
yarn cli service insert --name VRF
yarn cli service insert --name DATA_FEED
yarn cli service insert --name REQUEST_RESPONSE

keygen_output=$(yarn cli vrf keygen)
sk=$(echo $keygen_output | awk -F'sk=' '{print $2}' | awk '{print $1}')
pk=$(echo $keygen_output | awk -F'pk=' '{print $2}' | awk '{print $1}')
pkX=$(echo $keygen_output | awk -F'pkX=' '{print $2}' | awk '{print $1}')
pkY=$(echo $keygen_output | awk -F'pkY=' '{print $2}' | awk '{print $1}')
keyHash=$(echo $keygen_output | awk -F'keyHash=' '{print $2}' | awk '{print $1}')
echo keyHash: $keyHash

yarn cli vrf insert --chain localhost --sk $sk --pk $pk --pkX $pkX --pkY $pkY --keyHash $keyHash

cd ../contracts/v0.1
npx hardhat node --hostname 127.0.0.1 --no-deploy &

cd ../..
node update-vrf-migration.js
node update-rr-migration.js

prepayment_output=$(yarn deploy:localhost:prepayment)
vrf_output=$(yarn deploy:localhost:vrf)
rr_output=$(yarn deploy:localhost:rr)

vrf_address=$(echo $vrf_output | awk '/deployed at/{print $(NF-3); exit}')
rr_address=$(echo $rr_output | awk '/deployed at/{print $(NF-3); exit}')

cd ../cli

yarn cli listener insert --chain localhost --service VRF --address $vrf_address --eventName RandomWordsRequested
yarn cli reporter insert --chain localhost --service VRF --address 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266 --privateKey 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 --oracleAddress $vrf_address

yarn cli listener insert --chain localhost --service REQUEST_RESPONSE --address $vrf_address --eventName DataRequested
yarn cli reporter insert --chain localhost --service REQUEST_RESPONSE --address 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266 --privateKey 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 --oracleAddress $vrf_address

cd ../core

yarn start:listener:vrf &
yarn start:worker:vrf &
yarn start:reporter:vrf &

yarn start:listener:request_response &
yarn start:worker:request_response &
yarn start:reporter:request_response &