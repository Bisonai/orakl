import { deployments, ethers, getNamedAccounts } from "hardhat";
import {
  getKeyHash,
  estimateVRFServiceFee,
  estimateRRServiceFee,
  getAggregatorPairs,
  priceFeeds,
  findPairsWithSameRoundId,
} from "./utils";

async function main() {
  const {
    sResponse: rrResponseBefore,
    sRandomWord: vrfResponseBefore,
    priceFeedResults: priceFeedsBefore,
  } = await read();
  console.log(priceFeedsBefore);
  await request();
  await new Promise((resolve) => setTimeout(resolve, 30000));
  const {
    sResponse: rrResponseAfter,
    sRandomWord: vrfResponseAfter,
    priceFeedResults: priceFeedsAfter,
  } = await read();
  if (rrResponseBefore == rrResponseAfter) {
    throw "check if request response is alive";
  }
  if (vrfResponseBefore == vrfResponseAfter) {
    throw "check if vrf is alive";
  }
  const result = findPairsWithSameRoundId(priceFeedsBefore, priceFeedsAfter);
  if (result.length > 0) {
    throw `check following pairs:${result}`;
  }

  console.log("successful");
}

async function read() {
  const _inspectorConsumer = await deployments.get("InspectorConsumer");
  const inspectorConsumer = await ethers.getContractAt(
    _inspectorConsumer.abi,
    _inspectorConsumer.address
  );

  const sResponse = await inspectorConsumer.sResponse();
  const sRandomWord = await inspectorConsumer.sResponse();

  const priceFeedResults: priceFeeds = {};
  const pairs = await getAggregatorPairs();
  for (const pair of pairs) {
    const priceFeedResult = await inspectorConsumer.requestDataFeed(pair);
    priceFeedResults[pair] = {
      roundId: priceFeedResult[0],
      answer: priceFeedResult[1],
    };
  }
  return { sResponse, sRandomWord, priceFeedResults };
}

async function request() {
  const _inspectorConsumer = await deployments.get("InspectorConsumer");
  const inspectorConsumer = await ethers.getContractAt(
    _inspectorConsumer.abi,
    _inspectorConsumer.address
  );

  const keyHash = getKeyHash();
  const callbackGasLimit = 500_000;
  const numWords = 1;

  const { deployer } = await getNamedAccounts();
  const estimatedVRFServiceFee = await estimateVRFServiceFee();

  let txReceipt = await (
    await inspectorConsumer.requestVRFDirect(
      keyHash,
      callbackGasLimit,
      numWords,
      deployer,
      {
        value: ethers.parseEther(estimatedVRFServiceFee.toString()),
      }
    )
  ).wait();

  console.log("Requested random words using direct payment");

  const estimatedRRServiceFee = await estimateRRServiceFee();
  txReceipt = await (
    await inspectorConsumer.requestRRDirect(callbackGasLimit, {
      value: ethers.parseEther(estimatedRRServiceFee.toString()),
    })
  ).wait();

  console.log("Requested data using direct payment");
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
