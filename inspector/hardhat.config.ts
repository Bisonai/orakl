import { HardhatUserConfig, task } from "hardhat/config";
import { HardhatRuntimeEnvironment } from "hardhat/types";
import {
  readAll,
  readVrf,
  readRr,
  requestAll,
  requestVrf,
  requestRr,
} from "./scripts/utils";
import "@nomicfoundation/hardhat-toolbox";
import "hardhat-deploy";
import dotenv from "dotenv";

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
      url:
        process.env.PROVIDER ||
        "https://klaytn-baobab-rpc.allthatnode.com:8551",
      chainId: 1001,
      ...commonConfig,
      gasPrice: 250_000_000_000,
    },
    cypress: {
      url: process.env.PROVIDER || "https://public-en-cypress.klaytn.net",
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

task("inspect", "Main task")
  .addParam("service", "The service to use", "all")
  .setAction(async (taskArgs, hre: HardhatRuntimeEnvironment) => {
    const { ethers, deployments, network } = hre;
    const { service } = taskArgs;

    const ACC_ID =
      network.name == "baobab"
        ? (process.env.BAOBAB_ACC_ID as string)
        : (process.env.CYPRESS_ACC_ID as string);
    const explorerBaseUrl =
      network.name == "baobab"
        ? "https://baobab.klaytnfinder.io/tx"
        : "https://klaytnfinder.io/tx";

    const _inspectorConsumer = await deployments.get("InspectorConsumer");
    const inspectorConsumer = await ethers.getContractAt(
      _inspectorConsumer.abi,
      _inspectorConsumer.address
    );

    if (service == "rr") {
      const { sResponse: rrResponseBefore, rrRequestId: rrRequestIdBefore } =
        await readRr(inspectorConsumer);
      await requestRr(ACC_ID, explorerBaseUrl, inspectorConsumer);
      await new Promise((resolve) => setTimeout(resolve, 5000));
      const { sResponse: rrResponseAfter, rrRequestId: rrRequestIdAfter } =
        await readRr(inspectorConsumer);

      if (
        rrResponseBefore == rrResponseAfter ||
        rrRequestIdBefore == rrRequestIdAfter
      ) {
        console.error("RR fulfillment: FAILURE");
      } else {
        console.log("RR fulfillment: SUCCESS");
      }
    } else if (service == "vrf") {
      const {
        sRandomWord: vrfResponseBefore,
        vrfRequestId: vrfRequestIdBefore,
      } = await readVrf(inspectorConsumer);
      await requestVrf(
        ACC_ID,
        explorerBaseUrl,
        inspectorConsumer,
        network.name
      );
      await new Promise((resolve) => setTimeout(resolve, 5000));
      const { sRandomWord: vrfResponseAfter, vrfRequestId: vrfRequestIdAfter } =
        await readVrf(inspectorConsumer);

      if (
        vrfResponseBefore == vrfResponseAfter ||
        vrfRequestIdBefore == vrfRequestIdAfter
      ) {
        console.error("VRF fulfillment: FAILURE");
      } else {
        console.log("VRF fulfillment: SUCCESS");
      }
    } else {
      const {
        sResponse: rrResponseBefore,
        sRandomWord: vrfResponseBefore,
        rrRequestId: rrRequestIdBefore,
        vrfRequestId: vrfRequestIdBefore,
      } = await readAll(inspectorConsumer);
      await requestAll(
        ACC_ID,
        explorerBaseUrl,
        inspectorConsumer,
        network.name
      );
      await new Promise((resolve) => setTimeout(resolve, 5000));
      const {
        sResponse: rrResponseAfter,
        sRandomWord: vrfResponseAfter,
        rrRequestId: rrRequestIdAfter,
        vrfRequestId: vrfRequestIdAfter,
      } = await readAll(inspectorConsumer);

      if (
        rrResponseBefore == rrResponseAfter ||
        rrRequestIdBefore == rrRequestIdAfter
      ) {
        console.error("RR fulfillment: FAILURE");
      } else {
        console.log("RR fulfillment: SUCCESS");
      }

      if (
        vrfResponseBefore == vrfResponseAfter ||
        vrfRequestIdBefore == vrfRequestIdAfter
      ) {
        console.error("VRF fulfillment: FAILURE");
      } else {
        console.log("VRF fulfillment: SUCCESS");
      }
    }
  });

export default config;
