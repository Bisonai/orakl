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

    // Add RequestResponseConsumer as consumer of Prepayment account
    const _inspectorConsumer = await deployments.get("InspectorConsumer");
    const consumerAddress = _inspectorConsumer.address;
    await (await prepayment.addConsumer(ACC_ID, consumerAddress)).wait();
    console.log(
      `Added RequestResponseConsumer ${consumerAddress} to prepayment account`
    );
  } else {
    console.log(`Prepayment account ID not defined`);
  }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
