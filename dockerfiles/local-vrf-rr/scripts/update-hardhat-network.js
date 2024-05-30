const fs = require("node:fs");
const path = require("node:path");

const dir = "contracts/v0.1/hardhat.config.cjs";
let fileContent = fs.readFileSync(dir, "utf8");
let pattern = /(localhost: {\n\s*gasPrice: 250_000_000_000\n\s*},)/g;
let replacement = `localhost: {
    gasPrice: 250_000_000_000,
    url: 'http://json-rpc:8545'
  },`;
fileContent = fileContent.replace(pattern, replacement);
fs.writeFileSync(dir, fileContent);
