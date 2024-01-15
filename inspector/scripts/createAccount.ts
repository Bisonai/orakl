import { ethers, getNamedAccounts, network } from "hardhat";
import { Prepayment__factory } from "@bisonai/orakl-contracts";
import dotenv from "dotenv";

dotenv.config();

async function main() {
  let ACC_ID;
  if (network.name == "baobab") {
    ACC_ID = process.env.BAOBAB_ACC_ID;
  } else {
    ACC_ID = process.env.CYPRESS_ACC_ID;
  }

  if (!ACC_ID) {
    console.log("Generating new prepayment account ID");
    const { prepayment: prepaymentAddress } = await getNamedAccounts();
    const prepayment = await ethers.getContractAt(
      [...Prepayment__factory.abi],
      prepaymentAddress
    );

    // Create a new account. One address can make many accounts.
    const txReceipt = await (await prepayment.createAccount()).wait();
    console.log(txReceipt.logs);
  } else {
    console.log(`Prepayment account ID already defined: ${ACC_ID}`);
  }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
