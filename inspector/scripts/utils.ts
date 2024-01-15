import hre from "hardhat";
import { ethers } from "hardhat";
import { CoordinatorBase__factory } from "@bisonai/orakl-contracts";
import dotenv from "dotenv";
import axios from "axios";
import { JSDOM } from "jsdom";

dotenv.config();

export type priceFeeds = {
  [key: string]: {
    roundId: bigint;
    answer: bigint;
  };
};

export function getKeyHash() {
  const networkName = hre.network.name;
  if (networkName == "baobab") {
    return "0xd9af33106d664a53cb9946df5cd81a30695f5b72224ee64e798b278af812779c";
  } else if (networkName == "cypress") {
    return "0x6cff5233743b3c0321a19ae11ab38ae0ddc7ddfe1e91b162fa8bb657488fb157";
  } else {
    throw new Error(`Key Hash is not defined for network: ${networkName}`);
  }
}

export async function estimateVRFServiceFee() {
  const { vrfCoordinator: coordinatorAddress } = await hre.getNamedAccounts();
  const coordinator = await ethers.getContractAt(
    [...CoordinatorBase__factory.abi],
    coordinatorAddress
  );

  const reqCount = 1;
  const numSubmission = 1;
  const callbackGasLimit = 500_000;
  const estimatedServiceFee = await coordinator.estimateFee(
    reqCount,
    numSubmission,
    callbackGasLimit
  );
  const amountKlay = ethers.formatUnits(estimatedServiceFee, "ether");

  //console.log(`Estimated Price for 1 VRF Request is: ${amountKlay} Klay`);
  return amountKlay;
}

export async function estimateRRServiceFee() {
  const { rrCoordinator: coordinatorAddress } = await hre.getNamedAccounts();
  const coordinator = await ethers.getContractAt(
    [...CoordinatorBase__factory.abi],
    coordinatorAddress
  );

  const reqCount = 1;
  const numSubmission = 1;
  const callbackGasLimit = 500_000;
  const estimatedServiceFee = await coordinator.estimateFee(
    reqCount,
    numSubmission,
    callbackGasLimit
  );
  const amountKlay = ethers.formatUnits(estimatedServiceFee, "ether");

  //console.log(`Estimated Price for 1 RR Request is '${amountKlay}' Klay`);
  return amountKlay;
}

export async function getAggregatorPairs() {
  const url = "https://config.orakl.network/";
  const response = await axios.get(url);
  const dom = new JSDOM(response.data);
  const rows = dom.window.document.querySelectorAll(
    "table:nth-child(5) tbody tr"
  );
  const result: Set<string> = new Set();
  for (const row of rows) {
    const cell = row.querySelector("td:nth-child(1) a");
    const pairData = cell?.textContent?.trim();
    if (!pairData || ["PEG-POR", "MNR-KRW"].includes(pairData)) {
      continue;
    }
    result.add(pairData);
  }

  return Array.from(result);
}

export function findPairsWithSameRoundId(
  feeds0: priceFeeds,
  feeds1: priceFeeds
) {
  const results = [];
  for (const pair in feeds0) {
    if (feeds0[pair].roundId === feeds1[pair].roundId) {
      results.push(pair);
    }
  }
  return results;
}
