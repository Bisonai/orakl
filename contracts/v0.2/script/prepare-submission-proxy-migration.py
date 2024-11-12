import json
import argparse
import os
from datetime import datetime

file_path = "./addresses/datafeeds-addresses.json"

def prepareForSubmissionProxy(data, network):
    entries = []
    for symbol, networks in data.items():
        name = symbol
        for key, item in networks.items():
            if key == network:
                entries.append({
                    "name": name,
                    "address": item["feed"]
                })

    result = {
        "deploy": {},
        "addOracle": {
            "oracles": []
        },
        "updateFeed": entries
    }

    return result

def prepareForFeed(data, network):
    entries = []
    for _, networks in data.items():
        for key, item in networks.items():
            if key == network:
                entries.append(item["feed"])
    result = {
        "updateSubmitter": {
            "submitter": "",
            "feedAddresses": entries
        }
    }
    return result

def main():
    with open(file_path, "r") as f:
        data = json.load(f)

    parser = argparse.ArgumentParser(description="parse args")
    parser.add_argument('--network', type=str, default="baobab")
    parser.add_argument('--contract', type=str, default="SubmissionProxy")

    args = parser.parse_args()
    network = args.network
    contract = args.contract

    if contract == "SubmissionProxy":
        result = prepareForSubmissionProxy(data, network)
    elif contract == "Feed":
        result = prepareForFeed(data, network)
    else:
        raise ValueError("Invalid contract name")

    timestamp = datetime.now().strftime('%Y%m%d%H%M%S')

    # Create the file path with the timestamp
    write_file_path = f"./migration/{network}/{contract}/{timestamp}_update.json"

    # Ensure the directories exist
    os.makedirs(os.path.dirname(write_file_path), exist_ok=True)

    with open(write_file_path, 'w') as file:
        json.dump(result, file, indent=2)


if __name__ == "__main__":
    main()