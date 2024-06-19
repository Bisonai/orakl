const fs = require("node:fs");
const path = require("node:path");

const dir = "contracts/v0.1/migration/localhost/RequestResponse";
const address = process.argv[2];

const files = fs.readdirSync(dir);
const jsonFile = files.find((file) => path.extname(file) === ".json");
const filePath = path.join(dir, jsonFile);
const migrationData = JSON.parse(fs.readFileSync(filePath, "utf8"));
migrationData["registerOracle"] = [address];
fs.writeFileSync(filePath, JSON.stringify(migrationData, null, 2));
