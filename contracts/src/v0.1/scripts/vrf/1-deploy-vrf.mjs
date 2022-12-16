import pkg from 'hardhat'
const { ethers } = pkg

const ZERO_ADDRESS = ethers.constants.AddressZero

async function main() {
  const listen = false

  let VRFCoordinator = await ethers.getContractFactory('VRFCoordinator')
  VRFCoordinator = await VRFCoordinator.deploy()
  await VRFCoordinator.deployed()
  console.log('VRFCoordinator Address:', VRFCoordinator.address)
  
  //Register Proving Key

  const oracle = '0x45778c29A34bA00427620b937733490363839d8C' // Hardhat account 19
  const publicProvingKey = [
    '95162740466861161360090244754314042169116280320223422208903791243647772670481',
    '53113177277038648369733569993581365384831203706597936686768754351087979105423'
  ]
  await VRFCoordinator.registerProvingKey("0x72eBC7770884117fd2b6aA322A977a8Adb0527ee", publicProvingKey)

  // if (true || listen) {
  //   VRFCoordinator.once('ProvingKeyRegistered', async (keyHash, oracle) => {
  //     console.log(`keyHash ${keyHash}`)
  //     console.log(`oracle ${oracle}`)
  //   })
  // }

  // const minimumRequestConfirmations = 3
  // const maxGasLimit = 1_000_000
  // const gasAfterPaymentCalculation = 1_000_000

  // const feeConfig = {
  //   fulfillmentFlatFeeLinkPPMTier1: 0,
  //   fulfillmentFlatFeeLinkPPMTier2: 0,
  //   fulfillmentFlatFeeLinkPPMTier3: 0,
  //   fulfillmentFlatFeeLinkPPMTier4: 0,
  //   fulfillmentFlatFeeLinkPPMTier5: 0,
  //   reqsForTier2: 0,
  //   reqsForTier3: 0,
  //   reqsForTier4: 0,
  //   reqsForTier5: 0
  // }

  // // Configure VRF Coordinator
  // await VRFCoordinator.setConfig(
  //   minimumRequestConfirmations,
  //   maxGasLimit,
  //   gasAfterPaymentCalculation,
  //   feeConfig
  // )

  // if (listen) {
  //   VRFCoordinator.once(
  //     'ConfigSet',
  //     async (minimumRequestConfirmations, maxGasLimit, gasAfterPaymentCalculation, feeConfig) => {
  //       console.log(`minimumRequestConfirmations ${minimumRequestConfirmations}`)
  //       console.log(`maxGasLimit ${maxGasLimit}`)
  //       console.log(`gasAfterPaymentCalculation ${gasAfterPaymentCalculation}`)
  //       console.log(`feeConfig ${feeConfig}`)
  //     }
  //   )
  // }
  // // deploy consumer
  // let VRFConsumerMock = await ethers.getContractFactory('VRFConsumerMock')
  // VRFConsumerMock = await VRFConsumerMock.deploy(VRFCoordinator.address)
  // await VRFConsumerMock.deployed()
  // console.log('VRFConsumerMock Address:', VRFConsumerMock.address)

  // await VRFCoordinator.createSubscription()
  // if (listen) {
  //   VRFCoordinator.once('SubscriptionCreated', async (subId, owner) => {
  //     console.log('SubscriptionCreated')
  //     console.log(`subId ${subId}`)
  //     console.log(`owner ${owner}`)
  //   })
  // }

  // const subId = 1
  // await VRFCoordinator.addConsumer(subId, VRFConsumerMock.address)
  // if (listen) {
  //   await VRFCoordinator.once('SubscriptionConsumerAdded', async (subId, consumer) => {
  //     console.log('SubscriptionConsumerAdded')
  //     console.log(`subId ${subId}`)
  //     console.log(`consumer ${consumer}`)
  //   })
  // }
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
