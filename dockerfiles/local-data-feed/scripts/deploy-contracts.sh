rm -rf ./deployments/baobab_test
rm -rf ./migration/baobab_test

node ./scripts/v0.1/admin-aggregator/generate-aggregator-deployments.cjs --pairs [\"BTC-USDT\"] --chain baobab_test
jq --argjson input $(yarn hardhat deploy --network baobab_test --deploy-scripts deploy/Aggregator | tail -n 2 | head -n 1 ) '.address = $input["Aggregator_BTC-USDT"]' /app/samples/btc-usdt.aggregator.json > /app/samples/updated-btc-usdt.aggregator.json
jq '.bulk[0].adapterSource = "/app/samples/btc-usdt.adapter.json" | .bulk[0].aggregatorSource = "/app/samples/updated-btc-usdt.aggregator.json"' /app/contracts/scripts/v0.1/tmp/bulk.json > /app/contracts/scripts/v0.1/tmp/updated_bulk.json