const fs = require("node:fs");
const path = require("node:path");

const dir = "contracts/v0.1/migration/localhost/RequestResponse";

const files = fs.readdirSync(dir);
const jsonFile = files.find((file) => path.extname(file) === ".json");
const filePath = path.join(dir, jsonFile);
const migrationData = JSON.parse(fs.readFileSync(filePath, "utf8"));
migrationData["registerOracle"] = [
  "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
];
fs.writeFileSync(filePath, JSON.stringify(migrationData, null, 2));
