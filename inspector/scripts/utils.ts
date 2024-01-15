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
