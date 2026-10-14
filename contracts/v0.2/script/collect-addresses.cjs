// Collects deployed addresses from ./addresses/<chain>/*.json into
// ./addresses/feeds.json (data feeds) and ./addresses/core.json (FeedRouter, SubmissionProxy).
// datafeeds-addresses.json and others-addresses.json are deprecated copies kept for
// existing consumers of their raw GitHub URLs.
//
// Only pairs present in the live config (https://config.orakl.network/<chain>_feeds.json)
// are published, so removing a pair from orakl-config drops it here on next run.
//
// Usage:
//   node ./script/collect-addresses.cjs          # write files
//   node ./script/collect-addresses.cjs --check  # exit 1 if files are out of date
const path = require("node:path");
const fs = require("node:fs");

const CONFIG_BASE_URL = process.env.CONFIG_BASE_URL || "https://config.orakl.network";
const Chains = ["mainnet", "kairos"];
// Deprecated aliases kept in the output so existing consumers keep working.
const LegacyChainNames = { mainnet: "cypress", kairos: "baobab" };
const addressesPath = path.join(__dirname, "../addresses/");
const LegacyFileNames = {
  "feeds.json": "datafeeds-addresses.json",
  "core.json": "others-addresses.json",
};

const fetchConfig = async (chain) => {
  const url = `${CONFIG_BASE_URL}/${chain}_feeds.json`;
  const res = await fetch(url);
  if (!res.ok) {
    throw new Error(`Failed to fetch ${url}: ${res.status}`);
  }
  const data = await res.json();
  if (!Array.isArray(data) || data.length === 0) {
    throw new Error(`Unexpected config at ${url}`);
  }
  return data;
};

const toTag = (numFeeds) =>
  numFeeds > 8 ? "premium" : numFeeds > 5 ? "standard" : numFeeds > 1 ? "basic" : "single";

const convertContractType = (contractType) => {
  if (contractType == "Feed") {
    return "feed";
  } else if (contractType == "FeedProxy") {
    return "proxy";
  } else {
    return "unknown";
  }
};

const readAddresses = (chain) => {
  const dir = path.join(addressesPath, chain);
  const feeds = {};
  const others = {};

  for (const file of fs.readdirSync(dir).sort()) {
    if (path.extname(file) !== ".json") {
      continue;
    }
    const { address } = JSON.parse(fs.readFileSync(path.join(dir, file), "utf8"));
    let contractName = path.basename(file, ".json").replace(" ", "");

    if (contractName.startsWith("Feed") && contractName.includes("_")) {
      const [contractType, pairName] = contractName.split("_");
      feeds[pairName] = feeds[pairName] || {};
      feeds[pairName][convertContractType(contractType)] = address;
    } else {
      const splitted = contractName.split("_");
      if (splitted.length > 1) {
        splitted.pop();
        contractName = splitted.join("_");
      }
      others[contractName] = address;
    }
  }

  return { feeds, others };
};

const sortKeys = (obj) =>
  Object.fromEntries(Object.keys(obj).sort().map((k) => [k, obj[k]]));

const collect = async () => {
  const configs = {};
  for (const chain of Chains) {
    configs[chain] = new Map((await fetchConfig(chain)).map((c) => [c.name, c]));
  }

  const feeds = {};
  const others = {};

  for (const chain of Chains) {
    const deployed = readAddresses(chain);

    for (const [pairName, addresses] of Object.entries(deployed.feeds)) {
      if (!configs[chain].has(pairName)) {
        continue;
      }
      feeds[pairName] = feeds[pairName] || {};
      feeds[pairName][chain] = addresses;
    }

    others[chain] = sortKeys(deployed.others);
  }

  for (const [pairName, entry] of Object.entries(feeds)) {
    const config = configs.mainnet.get(pairName) || configs.kairos.get(pairName);
    const result = { tag: toTag(config.feeds.length) };
    for (const chain of Chains) {
      if (entry[chain]) {
        result[chain] = entry[chain];
      }
    }
    for (const chain of Chains) {
      if (entry[chain]) {
        result[LegacyChainNames[chain]] = entry[chain];
      }
    }
    feeds[pairName] = result;
  }

  for (const chain of Chains) {
    others[LegacyChainNames[chain]] = others[chain];
  }

  return { feeds: sortKeys(feeds), others };
};

async function main() {
  const check = process.argv.includes("--check");
  const { feeds, others } = await collect();

  const outputs = {
    "feeds.json": JSON.stringify(feeds, null, 2) + "\n",
    "core.json": JSON.stringify(others, null, 2) + "\n",
  };
  for (const [file, legacyFile] of Object.entries(LegacyFileNames)) {
    outputs[legacyFile] = outputs[file];
  }

  let stale = false;
  for (const [file, content] of Object.entries(outputs)) {
    const filePath = path.join(addressesPath, file);
    if (check) {
      const current = fs.existsSync(filePath) ? fs.readFileSync(filePath, "utf8") : "";
      if (current !== content) {
        console.error(`${file} is out of date; run node ./script/collect-addresses.cjs`);
        stale = true;
      }
    } else {
      fs.writeFileSync(filePath, content);
    }
  }

  if (stale) {
    process.exitCode = 1;
  }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
