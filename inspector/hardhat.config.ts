import { HardhatUserConfig } from "hardhat/config";
import "./inspector-task";

import "@nomicfoundation/hardhat-toolbox";
import dotenv from "dotenv";
import "hardhat-deploy";

dotenv.config();

let commonConfig = {};
if (process.env.PRIV_KEY) {
  commonConfig = {
    gas: 5_000_000,
    accounts: [process.env.PRIV_KEY],
  };
} else {
  commonConfig = {
    gas: 5_000_000,
    accounts: {
      mnemonic: process.env.MNEMONIC || "",
    },
  };
}

const config: HardhatUserConfig = {
  solidity: {
    version: "0.8.16",
    settings: {
      optimizer: {
        enabled: true,
        runs: 1000,
      },
    },
  },

  networks: {
    localhost: {
      gas: 1_400_000,
      gasPrice: 250_000_000_000,
    },
    baobab: {
      url: process.env.PROVIDER || "https://public-en-kairos.node.kaia.io",
      chainId: 1001,
      ...commonConfig,
      gasPrice: 250_000_000_000,
    },
    cypress: {
      url: process.env.PROVIDER || "https://public-en.node.kaia.io",
      ...commonConfig,
      gasPrice: 250_000_000_000,
    },
  },
  namedAccounts: {
    deployer: {
      default: 0,
    },
    prepayment: {
      baobab: "0x8d3A1663d10eEb0bC9C9e537e1BBeA69383194e7",
      cypress: "0xc2C88492Cf7e5240C3EB49353539E75336960600",
    },
    vrfCoordinator: {
      baobab: "0xDA8c0A00A372503aa6EC80f9b29Cc97C454bE499",
      cypress: "0x3F247f70DC083A2907B8E76635986fd09AA80EFb",
    },
    rrCoordinator: {
      baobab: "0x5fe8a7445bFDB2Cd6d9f1DcfB3B33D8c365FFdB0",
      cypress: "0x159F3BB6609B4C709F15823F3544032942106042",
    },
    aggregatorRouter: {
      baobab: "0xAF821aaaEdeF65b3bC1668c0b910c5b763dF6354",
      cypress: "0x16937CFc59A8Cd126Dc70A75A4bd3b78f690C861",
    },
  },
};

export default config;
