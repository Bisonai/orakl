#!/bin/bash

# Dependencies: jq, curl, gcloud-sdk

check_klay_sync() {
    local our_block
    our_block=$(get_our_klay_block)

    local public_block_hex
    public_block_hex=$(get_public_klay_block)
    public_block=$((16#$public_block_hex)) || { echo "[ERROR] Failed to convert hex to decimal"; exit 1; }

    # Calculate difference
    diff=$((our_block - public_block))

    # Check if the abs of difference is less than 10
    if [ ${diff#-} -lt 10 ]; then
	echo "[INFO] Klaytn is synchronized"
    else
	echo "[ERROR] Synchronization failed. Remain blocks to latest: $diff"
    fi
}

get_public_klay_block() {
    public_json_rpc="https://public-en-cypress.klaytn.net"

    # Request block number to public JSON-RPC node
    response=$(curl \
		   -H "Content-type: application/json" \
		   --data '{"jsonrpc":"2.0","method":"klay_blockNumber","params":[],"id":83}' \
		   $public_json_rpc)

    # Get block number from response
    # Print result field without "0x" in json response
    public_klay_blockNumber=$(echo "$response" | jq -r '.result' | awk '{ sub("0x", ""); print $0 }')

    echo "$public_klay_blockNumber"
}

get_our_klay_block() {
    # Get log from JSON-RPC node
    response=$(gcloud compute ssh orakl-cypress-prod-node \
		      --project=orakl-cypress-prod \
		      --tunnel-through-iap \
		      -- -t tail -1 /data/pruning/log/kend.out)

    # Get block number from log
    # If col 8 string starts with "number=",
    # then print the value only integer part
    our_klay_blockNumber=$(echo "$response" | awk '$8 ~ /^number*/ { sub(/^number=/, "", $8); print $8 }')

    echo "$our_klay_blockNumber"
}

check_klay_sync
