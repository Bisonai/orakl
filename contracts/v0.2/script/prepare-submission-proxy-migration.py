import json
import argparse
import os
from datetime import datetime

file_path = "./addresses/datafeeds-addresses.json"

def main():
    with open(file_path, "r") as f:
        data = json.load(f)


    parser = argparse.ArgumentParser(description="parse args")
    parser.add_argument('--network', type=str, default="baobab")

    args = parser.parse_args()
    network = args.network

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

    timestamp = datetime.now().strftime('%Y%m%d%H%M%S')

    # Create the file path with the timestamp
    write_file_path = f"./migration/{network}/SubmissionProxy/{timestamp}_update.json"

    print(write_file_path)

    # Ensure the directories exist
    os.makedirs(os.path.dirname(write_file_path), exist_ok=True)

    with open(write_file_path, 'w') as file:
        json.dump(result, file, indent=2)


if __name__ == "__main__":
    main()