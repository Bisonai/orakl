const path = require("path");
const fs = require("fs");
const axios = require("axios");
const { writeFileSync, existsSync, mkdirSync } = require("node:fs");
const { readFile } = require("node:fs/promises");

const ValidChains = ["baobab", "cypress"];
const addressesPath = path.join(__dirname, "../addresses/");

const fetchTags = async () => {
  const url = "https://config.orakl.network/cypress_configs.json";
  let tags = {};

  try {
    const res = await axios.get(url);
    (res.data || []).forEach((feed) => {
      const numFeeds = feed.feeds.length;
      const tag =
        numFeeds > 8
          ? "premium"
          : numFeeds > 5
          ? "standard"
          : numFeeds == 1
          ? "single"
          : "basic";
      tags[feed.name] = tag;
    });
  } catch (error) {
    console.error(`Error fetching tags: ${error}`);
  }

  return tags;
};

const readDeployments = async (folderPath, tags) => {
  const feeds = {};
  const others = {};

  const readFolder = async (_folderPath) => {
    const files = await fs.promises.readdir(_folderPath);
    await Promise.all(
      files.map(async (file) => {
        const filePath = path.join(_folderPath, file);
        const stats = await fs.promises.stat(filePath);

        if (stats.isDirectory()) {
          await readFolder(filePath);
        } else if (isValidPath(filePath)) {
          const network = filePath.replace(addressesPath, "").split("/")[0];
          if (!ValidChains.includes(network)) {
            return;
          }

          const contractName = path.basename(file, ".json");
          const address = await readAddressFromFile(filePath);
          if (contractName.startsWith("Feed") && contractName.includes("_")) {
            const splitted = contractName.replace(" ", "").split("_");
            const contractType = splitted[0];
            const pairName = splitted[1];

            ensureObjectPath(feeds, [pairName, network]);

            feeds[pairName][network][convertContractType(contractType)] =
              address;
            feeds[pairName]["tag"] = tags[pairName];
          } else {
            ensureObjectPath(others, [network]);

            const splitted = contractName.replace(" ", "").split("_");
            if (splitted.length > 1) {
              splitted.pop();
              contractName = splitted.join("_");
            }
            others[network][contractName] = address;
          }
        }
      })
    );
  };

  try {
    await readFolder(folderPath);
  } catch (error) {
    console.error(`Error reading deployments: ${error}`);
  }

  return { feeds, others };
};

const ensureObjectPath = (object, path) => {
  path.forEach((key) => {
    if (!object[key]) {
      object[key] = {};
    }
    object = object[key];
  });
};

const isValidPath = (_path) => {
  const fileName = path.basename(_path);
  return path.extname(fileName) === ".json";
};

const readAddressFromFile = async (filePath) => {
  const json = await readFile(filePath, "utf8");
  const data = JSON.parse(json);
  return data.address;
};

const convertContractType = (contractType) => {
  if (contractType == "Feed") {
    return "feed";
  } else if (contractType == "FeedProxy") {
    return "proxy";
  } else {
    return "unknown";
  }
};

async function main() {
  const tags = await fetchTags();
  const { feeds, others } = await readDeployments(addressesPath, tags);
  writeFileSync(
    path.join(addressesPath, "datafeeds-addresses.json"),
    JSON.stringify(feeds, null, 2)
  );
  writeFileSync(
    path.join(addressesPath, "others-addresses.json"),
    JSON.stringify(others, null, 2)
  );
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
