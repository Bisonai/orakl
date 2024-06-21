import { task } from "hardhat/config";
import { HardhatRuntimeEnvironment } from "hardhat/types";
import {
  readAll,
  readVrf,
  readRr,
  requestAll,
  requestVrf,
  requestRr,
  getKeyHash,
} from "./scripts/utils";
import { Prepayment__factory } from "@bisonai/orakl-contracts";
import "@nomiclabs/hardhat-ethers";

task("inspect", "Main task")
  .addParam("service", "The service to use", "all")
  .setAction(async (taskArgs, hre: HardhatRuntimeEnvironment) => {
    const { ethers, deployments, network } = hre;
    const { service } = taskArgs;

    const accId =
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

    const readAndRequest = async (readFn: Function, requestFn: Function) => {
      const before = await readFn(inspectorConsumer);
      await requestFn(accId, explorerBaseUrl, inspectorConsumer, network.name);
      await new Promise((resolve) => setTimeout(resolve, 5000));
      const after = await readFn(inspectorConsumer);

      return { before, after };
    };

    const checkFulfillment = (
      before: any,
      after: any,
      requestIdKey: string,
      responseKey: string,
      serviceName: string
    ) => {
      if (
        before[responseKey] == after[responseKey] ||
        before[requestIdKey] == after[requestIdKey]
      ) {
        console.error(`${serviceName} fulfillment: FAILURE`);
      } else {
        console.log(`${serviceName} fulfillment: SUCCESS`);
      }
    };

    if (service == "rr") {
      const { before, after } = await readAndRequest(readRr, requestRr);

      checkFulfillment(before, after, "rrRequestId", "sResponse", "RR");
    } else if (service == "vrf") {
      const { before, after } = await readAndRequest(readVrf, requestVrf);

      checkFulfillment(before, after, "vrfRequestId", "sRandomWord", "VRF");
    } else {
      const { before, after } = await readAndRequest(readAll, requestAll);

      checkFulfillment(before, after, "rrRequestId", "sResponse", "RR");
      checkFulfillment(before, after, "vrfRequestId", "sRandomWord", "VRF");
    }
  });

task("load-test-vrf", "Load test vrf task")
  .addOptionalParam("batch", "Batch size")
  .setAction(async (taskArgs, hre: HardhatRuntimeEnvironment) => {
    try {
      const { ethers, deployments, network } = hre;
      const batch = Number(taskArgs.batch) || 1;

      const accId =
        network.name == "baobab"
          ? (process.env.BAOBAB_ACC_ID as string)
          : (process.env.CYPRESS_ACC_ID as string);

      const _loadTestVRFConsumer = await deployments.get("LoadTestVRFConsumer");
      const loadTestVRFConsumer = await ethers.getContractAt(
        _loadTestVRFConsumer.abi,
        _loadTestVRFConsumer.address
      );

      const keyHash = getKeyHash(network.name);
      const callbackGasLimit = 500_000;
      const numWords = 1;

      console.log(
        `Batch size: ${batch}. Each batch contains 50 requests. Total requests: ${
          50 * batch
        }`
      );
      for (let i = 0; i < batch; i++) {
        await (
          await loadTestVRFConsumer.requestVRF(
            keyHash,
            accId,
            callbackGasLimit,
            numWords
          )
        ).wait();
      }

      const count = 50 * batch;
      let len = 0;

      while (len < count) {
        len = await loadTestVRFConsumer.getBlockRecordsLength();
      }

      const blockRecords = [];
      for (let i = 0; i < count; i++) {
        blockRecords.push(await loadTestVRFConsumer.blockRecords(i));
      }

      let blocks = "";
      blockRecords
        .map((block: bigint) => Number(block))
        .sort((a, b) => a - b)
        .map((block) => (blocks = blocks + block + " "));

      console.log(
        `Number of blocks/seconds it took to fulfill ${count} requests, in ascending order: `
      );
      console.log(blocks);

      await loadTestVRFConsumer.clear();
    } catch (e) {
      console.error(e);
    }
  });

task("load-test-rr", "Load test task")
  .addOptionalParam("batch", "Batch size")
  .setAction(async (taskArgs, hre: HardhatRuntimeEnvironment) => {
    try {
      const { ethers, deployments, network } = hre;
      const batch = Number(taskArgs.batch) || 1;

      const accId =
        network.name == "baobab"
          ? (process.env.BAOBAB_ACC_ID as string)
          : (process.env.CYPRESS_ACC_ID as string);

      const _loadTestRRConsumer = await deployments.get("LoadTestRRConsumer");
      const loadTestRRConsumer = await ethers.getContractAt(
        _loadTestRRConsumer.abi,
        _loadTestRRConsumer.address
      );

      const callbackGasLimit = 500_000;

      console.log(
        `Batch size: ${batch}. Each batch contains 50 requests. Total requests: ${
          50 * batch
        }`
      );
      for (let i = 0; i < batch; i++) {
        await (
          await loadTestRRConsumer.requestRR(accId, callbackGasLimit)
        ).wait();
      }

      const count = 50 * batch;
      let len = 0;

      while (len < count) {
        len = await loadTestRRConsumer.getBlockRecordsLength();
      }

      const blockRecords = [];
      for (let i = 0; i < count; i++) {
        blockRecords.push(await loadTestRRConsumer.blockRecords(i));
      }

      let blocks = "";
      blockRecords
        .map((block: bigint) => Number(block))
        .sort((a, b) => a - b)
        .map((block) => (blocks = blocks + block + " "));

      console.log(
        `Number of blocks/seconds it took to fulfill ${count} requests, in ascending order: `
      );
      console.log(blocks);

      await loadTestRRConsumer.clear();
    } catch (e) {
      console.error(e);
    }
  });

task("addConsumer", "Add consumer")
  .addOptionalParam("consumer", "Consumer Contract Name")
  .setAction(async (taskArgs, hre) => {
    const { deployments } = hre;
    const accId =
      network.name === "baobab"
        ? process.env.BAOBAB_ACC_ID
        : process.env.CYPRESS_ACC_ID;

    // consumer options are: "inspector", "vrf", "rr"
    let consumerContractName;
    switch (taskArgs.consumer) {
      case "inspector":
        consumerContractName = "InspectorConsumer";
        break;
      case "vrf":
        consumerContractName = "LoadTestVRFConsumer";
        break;
      case "rr":
        consumerContractName = "LoadTestRRConsumer";
        break;
      default:
        consumerContractName = "InspectorConsumer";
    }
    const consumerAddress = (await deployments.get(consumerContractName))
      .address;

    if (accId && consumerAddress) {
      const { prepayment: prepaymentAddress } = await hre.getNamedAccounts();
      const prepayment = await ethers.getContractAt(
        Prepayment__factory.abi,
        prepaymentAddress
      );
      await (await prepayment.addConsumer(accId, consumerAddress)).wait();

      console.log(`Added consumer ${consumerAddress} to prepayment account`);
    } else {
      if (!accId) console.error(`Prepayment accountId is not defined`);
      if (!consumerAddress) console.error(`Consumer Address is not defined`);
    }
  });
