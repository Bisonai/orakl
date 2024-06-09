#!/bin/bash

cd cli || exit

yarn cli chain insert --name localhost
yarn cli service insert --name VRF
yarn cli service insert --name REQUEST_RESPONSE

cd .. || exit
node update-rr-migration.js
node update-hardhat-network.js

cd contracts/v0.1 || exit
yarn deploy:localhost:prepayment
yarn deploy:localhost:rr

cd ../../cli || exit
yarn cli listener insert --chain localhost --service REQUEST_RESPONSE --address 0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512 --eventName DataRequested
yarn cli reporter insert --chain localhost --service REQUEST_RESPONSE --address 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266 --privateKey 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 --oracleAddress 0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512

tail -f /dev/null