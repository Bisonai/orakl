const fs = require("node:fs");
const path = require("node:path");

const dir = "contracts/v0.1/migration/localhost/VRF";
const pkX = process.argv[2];
const pkY = process.argv[3];

const files = fs.readdirSync(dir);
const jsonFile = files.find((file) => path.extname(file) === ".json");
const filePath = path.join(dir, jsonFile);
const migrationData = JSON.parse(fs.readFileSync(filePath, "utf8"));
migrationData["registerOracle"] = [
  {
    address: "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
    publicProvingKey: [pkX, pkY],
  },
];
fs.writeFileSync(filePath, JSON.stringify(migrationData, null, 2));
