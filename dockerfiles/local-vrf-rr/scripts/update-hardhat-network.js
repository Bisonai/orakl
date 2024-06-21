const fs = require("node:fs");

const providerUrl = process.argv[2];

const dir = "contracts/v0.1/hardhat.config.cjs";
let fileContent = fs.readFileSync(dir, "utf8");
const pattern = /(localhost: {\n\s*gasPrice: 250_000_000_000\n\s*},)/g;
const replacement = `localhost: {
    gasPrice: 250_000_000_000,
    url: ${providerUrl}
  },`;
fileContent = fileContent.replace(pattern, replacement);
fs.writeFileSync(dir, fileContent);
