module.exports = {
  "contracts/v0.2/**/*.sol": "yarn contracts-v02 lint",

  "core/**/*": "yarn core test test",
  "core/src/**/*.ts": "yarn core lint",
  "core/test/**/*.ts": "yarn core lint",
};
