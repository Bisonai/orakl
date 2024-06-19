const fs = require("node:fs");
const path = require("node:path");

const dir = "contracts/v0.1/migration/localhost/VRF";
const pkX = process.argv[2];
const pkY = process.argv[3];
const address = process.argv[4];

const files = fs.readdirSync(dir);
const jsonFile = files.find((file) => path.extname(file) === ".json");
const filePath = path.join(dir, jsonFile);
const migrationData = JSON.parse(fs.readFileSync(filePath, "utf8"));
migrationData["registerOracle"] = [
  {
    address: address,
    publicProvingKey: [pkX, pkY],
  },
];
fs.writeFileSync(filePath, JSON.stringify(migrationData, null, 2));
