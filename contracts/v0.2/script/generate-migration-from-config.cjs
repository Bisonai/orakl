const path = require("path");
const moment = require("moment");
const { parseArgs } = require("node:util");
const { writeFileSync, existsSync, mkdirSync } = require("node:fs");
const axios = require("axios");

const readArgs = async () => {
  const requiredArgs = ["--chain"];
  const options = { chain: { type: "string" } };
  const { values } = parseArgs({ requiredArgs, options });
  return values;
};

const fetchConfigData = async (chain) => {
  const url = `https://config.orakl.network/${chain}_configs.json`;
  try {
    const { data } = await axios.get(url);
    return data.map((config) => config.name);
  } catch (error) {
    console.error(`Failed to fetch config data from ${url}:`, error);
    throw error;
  }
};

const createResultObject = (names) => ({
  deploy: {},
  deployFeed: {
    feedNames: names,
  },
});

const ensureDirectoryExistence = (dirPath) => {
  if (!existsSync(dirPath)) {
    mkdirSync(dirPath, { recursive: true });
  }
};

const writeToFile = (filePath, data) => {
  writeFileSync(filePath, JSON.stringify(data, null, 2));
};

async function main() {
  let { chain } = await readArgs();
  chain = chain || "test";

  const names = await fetchConfigData(chain);
  const result = createResultObject(names);

  const date = moment().format("YYYYMMDDHHMMSS");
  const fileName = `${date}_deploy.json`;

  const savePath = path.join(
    __filename,
    `../../migration/${chain}/SubmissionProxy`
  );
  ensureDirectoryExistence(savePath);

  const filePath = path.join(savePath, fileName);
  writeToFile(filePath, result);
  console.log(`Migration file saved to: ${filePath}`);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
