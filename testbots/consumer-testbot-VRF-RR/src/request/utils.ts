import { ethers } from "ethers";
import { add0x } from "../utils";
import * as dotenv from "dotenv";
dotenv.config();

const PRIVATE_KEY_ENV = process.env.PRIVATE_KEY;
const PROVIDER_ENV = process.env.PROVIDER_URL;
export function buildWallet() {
  try {
    const { PRIVATE_KEY, PROVIDER } = checkParameters();
    const provider = new ethers.providers.JsonRpcProvider(PROVIDER);
    const wallet = new ethers.Wallet(PRIVATE_KEY, provider);
    return wallet;
  } catch (e) {
    console.error(e);
  }
}

function checkParameters() {
  if (!PRIVATE_KEY_ENV) {
    throw "Missing Private key";
  }

  if (!PROVIDER_ENV) {
    throw "Missing Provider";
  }

  return { PRIVATE_KEY: PRIVATE_KEY_ENV, PROVIDER: PROVIDER_ENV };
}

export async function sendTransaction(wallet, to, payload, gasLimit?, value?) {
  const tx = {
    from: wallet.address,
    to: to,
    data: add0x(payload),
    value: value || "0x00",
  };

  if (gasLimit) {
    tx["gasLimit"] = gasLimit;
  }

  console.debug("sendTransaction:tx");
  const txReceipt = await wallet.sendTransaction(tx);
  return txReceipt;
}
