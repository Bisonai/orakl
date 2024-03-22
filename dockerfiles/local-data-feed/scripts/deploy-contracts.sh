#!/bin/bash
rm -rf ./deployments/baobab_test
rm -rf ./migration/baobab_test

node ./scripts/admin-aggregator/generate-aggregator-deployments.cjs --pairs [\"BTC-USDT\"] --chain baobab_test

curl -o "/app/contracts/v0.1/scripts/tmp/btc-usdt.aggregator.json" "https://config.orakl.network/aggregator/baobab/btc-usdt.aggregator.json"

jq --argjson input "$(yarn hardhat deploy --network baobab_test --deploy-scripts deploy/Aggregator | tail -n 2 | head -n 1)" '.address = $input["Aggregator_BTC-USDT"]' /app/contracts/v0.1/scripts/tmp/btc-usdt.aggregator.json > /app/contracts/v0.1/scripts/tmp/updated-btc-usdt.aggregator.json
jq '.bulk[0].aggregatorSource = "/app/tmp/updated-btc-usdt.aggregator.json" | .bulk[0].adapterSource = "https://config.orakl.network/adapter/baobab/btc-usdt.adapter.json"' "/app/contracts/v0.1/scripts/tmp/bulk.json" > "/app/contracts/v0.1/scripts/tmp/updated_bulk.json"
