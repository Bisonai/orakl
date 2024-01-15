import { deployments, ethers, getNamedAccounts, network } from "hardhat";
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

  if (ACC_ID) {
    const { prepayment: prepaymentAddress } = await getNamedAccounts();
    const prepayment = await ethers.getContractAt(
      [...Prepayment__factory.abi],
      prepaymentAddress
    );

    // Deposit 5 KLAY
    const klayAmount = "5";
    const amount = ethers.parseEther(klayAmount);
    await (await prepayment.deposit(ACC_ID, { value: amount })).wait();
    console.log(`Deposited ${klayAmount} KLAY to account ID ${ACC_ID}`);
  } else {
    console.log(`Prepayment account ID not defined`);
  }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
