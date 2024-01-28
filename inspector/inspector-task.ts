import { task } from "hardhat/config";
import { HardhatRuntimeEnvironment } from "hardhat/types";
import {
  readAll,
  readVrf,
  readRr,
  requestAll,
  requestVrf,
  requestRr,
} from "./scripts/utils";

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
