#!/bin/bash

# Dependencies: jq, curl

check_klay_sync_baobab() {
    our_json_rpc="http://100.93.31.29:8551"
    our_block_hex=$(get_our_klay_block $our_json_rpc)
    our_block=$((16#$our_block_hex)) || { echo "[ERROR] Failed to convert hex to decimal from our block in check_klay_sync_baobab()"; exit 1; }

    public_json_rpc="https://public-en.kairos.node.kaia.io"
    public_block_hex=$(get_public_klay_block $public_json_rpc)
    public_block=$((16#$public_block_hex)) || { echo "[ERROR] Failed to convert hex to decimal from public block in check_klay_sync_baobab()"; exit 1; }

    # Calculate difference
    diff=$((our_block - public_block))

    # Check if the abs of difference is less than 10
    if [ ${diff#-} -lt 10 ]; then
	echo "[INFO] Baobab json-rpc is synchronized"
    else
	echo "[ERROR] Synchronization failed in baobab. Remaining blocks to the latest: $diff"
    fi
}

check_klay_sync_cypress() {
    our_json_rpc="http://100.75.43.49:8551"
    our_block_hex=$(get_our_klay_block $our_json_rpc)
    our_block=$((16#$our_block_hex)) || { echo "[ERROR] Failed to convert hex to decimal from our block in check_klay_sync_cypress()"; exit 1; }

    public_json_rpc="https://public-en-cypress.klaytn.net"
    public_block_hex=$(get_public_klay_block $public_json_rpc)
    public_block=$((16#$public_block_hex)) || { echo "[ERROR] Failed to convert hex to decimal from public block in check_klay_sync_cypress()"; exit 1; }

    # Calculate difference
    diff=$((our_block - public_block))

    # Check if the abs of difference is less than 10
    if [ ${diff#-} -lt 10 ]; then
	echo "[INFO] Cypress json-rpc is synchronized"
    else
	echo "[ERROR] Synchronization failed in cypress. Remaining blocks to the latest: $diff"
    fi
}

get_public_klay_block() {
    public_json_rpc=$1

    # Request block number to public JSON-RPC node
    response=$(curl \
		   -H "Content-type: application/json" \
		   --data '{"jsonrpc":"2.0","method":"klay_blockNumber","params":[],"id":83}' \
           -s \
		   $public_json_rpc)

    # Get block number from response
    # Print result field without "0x" in json response
    public_klay_blockNumber=$(echo "$response" | jq -r '.result' | awk '{ sub("0x", ""); print $0 }')

    echo "$public_klay_blockNumber"
}

get_our_klay_block() {
    our_json_rpc=$1

    # Request block number to public JSON-RPC node
    response=$(curl \
		   -H "Content-type: application/json" \
		   --data '{"jsonrpc":"2.0","method":"klay_blockNumber","params":[],"id":1}' \
           -s \
		   $our_json_rpc)

    # Get block number from response
    # Print result field without "0x" in json response
    our_klay_blockNumber=$(echo "$response" | jq -r '.result' | awk '{ sub("0x", ""); print $0 }')

    echo "$our_klay_blockNumber"
}

check_klay_sync_baobab
check_klay_sync_cypress
