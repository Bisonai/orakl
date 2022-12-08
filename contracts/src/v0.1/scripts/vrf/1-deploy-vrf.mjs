import pkg from 'hardhat'
const { ethers } = pkg

const ZERO_ADDRESS = ethers.constants.AddressZero

async function main() {
  let VRFCoordinator = await ethers.getContractFactory('VRFCoordinator')
  // const blockhashStore = ZERO_ADDRESS // FIXME
  VRFCoordinator = await VRFCoordinator.deploy(/* blockhashStore */)
  await VRFCoordinator.deployed()
  console.log('VRFCoordinator Address:', VRFCoordinator.address)

  // Register Proving Key
  const oracle = ZERO_ADDRESS // FIXME
  const publicProvingKey = [1, 2] // FIXME
  await VRFCoordinator.registerProvingKey(oracle, publicProvingKey)
  VRFCoordinator.once('ProvingKeyRegistered', async (keyHash, oracle) => {
    console.log(`keyHash ${keyHash}`)
    console.log(`oracle ${oracle}`)
  })

  const minimumRequestConfirmations = 3
  const maxGasLimit = 1_000_000
  const gasAfterPaymentCalculation = 1_000_000

  const feeConfig = {
    fulfillmentFlatFeeLinkPPMTier1: 0,
    fulfillmentFlatFeeLinkPPMTier2: 0,
    fulfillmentFlatFeeLinkPPMTier3: 0,
    fulfillmentFlatFeeLinkPPMTier4: 0,
    fulfillmentFlatFeeLinkPPMTier5: 0,
    reqsForTier2: 0,
    reqsForTier3: 0,
    reqsForTier4: 0,
    reqsForTier5: 0
  }

  // Configure VRF Coordinator
  await VRFCoordinator.setConfig(
    minimumRequestConfirmations,
    maxGasLimit,
    gasAfterPaymentCalculation,
    feeConfig
  )

  VRFCoordinator.once(
    'ConfigSet',
    async (minimumRequestConfirmations, maxGasLimit, gasAfterPaymentCalculation, feeConfig) => {
      console.log(`minimumRequestConfirmations ${minimumRequestConfirmations}`)
      console.log(`maxGasLimit ${maxGasLimit}`)
      console.log(`gasAfterPaymentCalculation ${gasAfterPaymentCalculation}`)
      /* console.log(`feeConfig ${feeConfig}`) */
    }
  )

  let VRFConsumerMock = await ethers.getContractFactory('VRFConsumerMock')
  VRFConsumerMock = await VRFConsumerMock.deploy(VRFCoordinator.address)
  await VRFConsumerMock.deployed()
  console.log('VRFConsumerMock Address:', VRFConsumerMock.address)

  await VRFCoordinator.createSubscription()
  VRFCoordinator.once('SubscriptionCreated', async (subId, owner) => {
    console.log('SubscriptionCreated')
    console.log(`subId ${subId}`)
    console.log(`owner ${owner}`)

    await VRFCoordinator.addConsumer(subId, VRFConsumerMock.address)
    VRFCoordinator.once('SubscriptionConsumerAdded', async (subId, consumer) => {
      console.log('SubscriptionConsumerAdded')
      console.log(`subId ${subId}`)
      console.log(`consumer ${consumer}`)
    })
  })
}

main()
