import { utils } from 'ethers'
import pkg from 'hardhat'
const { ethers } = pkg

const ZERO_ADDRESS = ethers.constants.AddressZero
function parseEther(amount) {
  return ethers.utils.parseUnits(amount.toString(), 18);
}
async function main() {
  const listen = false

  let Prepayment = await ethers.getContractFactory('Prepayment')
  Prepayment = await Prepayment.deploy()
  await Prepayment.deployed()
  console.log('Prepayment Address:', Prepayment.address)


  let VRFCoordinator = await ethers.getContractFactory('VRFCoordinator1')
  VRFCoordinator = await VRFCoordinator.deploy(Prepayment.address);
  await VRFCoordinator.deployed()
  console.log('VRFCoordinator Address:', VRFCoordinator.address)

  // Register Proving Key
  const oracle = '0xc3d9a9c86093f7bd4660c498c31ECBc76aA1d044' // Hardhat account 19
  const publicProvingKey = [
    '95162740466861161360090244754314042169116280320223422208903791243647772670481',
    '53113177277038648369733569993581365384831203706597936686768754351087979105423'
  ]
  await VRFCoordinator.registerProvingKey(oracle, publicProvingKey)

  if (true || listen) {
    VRFCoordinator.once('ProvingKeyRegistered', async (keyHash, oracle) => {
      console.log(`keyHash ${keyHash}`)
      console.log(`oracle ${oracle}`)
    })
  }

  const minimumRequestConfirmations = 3
  const maxGasLimit = 1_000_000
  const gasAfterPaymentCalculation = 1_000

  const feeConfig = {
    fulfillmentFlatFeeLinkPPMTier1: 1,
    fulfillmentFlatFeeLinkPPMTier2: 0,
    fulfillmentFlatFeeLinkPPMTier3: 2,
    fulfillmentFlatFeeLinkPPMTier4: 3,
    fulfillmentFlatFeeLinkPPMTier5: 4,
    reqsForTier2: 1,
    reqsForTier3: 2,
    reqsForTier4: 3,
    reqsForTier5: 4
  }

  // Configure VRF Coordinator
  await VRFCoordinator.setConfig(
    minimumRequestConfirmations,
    maxGasLimit,
    gasAfterPaymentCalculation,
    feeConfig
  )

  if (listen) {
    VRFCoordinator.once(
      'ConfigSet',
      async (minimumRequestConfirmations, maxGasLimit, gasAfterPaymentCalculation, feeConfig) => {
        console.log(`minimumRequestConfirmations ${minimumRequestConfirmations}`)
        console.log(`maxGasLimit ${maxGasLimit}`)
        console.log(`gasAfterPaymentCalculation ${gasAfterPaymentCalculation}`)
        console.log(`feeConfig ${feeConfig}`)
      }
    )
  }

  let VRFConsumerMock = await ethers.getContractFactory('VRFConsumerMock')
  VRFConsumerMock = await VRFConsumerMock.deploy(VRFCoordinator.address)
  await VRFConsumerMock.deployed()
  console.log('VRFConsumerMock Address:', VRFConsumerMock.address)

  await Prepayment.createSubscription()
  if (listen) {
    Prepayment.once('SubscriptionCreated', async (subId, owner) => {
      console.log('SubscriptionCreated')
      console.log(`subId ${subId}`)
      console.log(`owner ${owner}`)
    })
  }

  const subId = 1
  await Prepayment.addConsumer(subId, VRFConsumerMock.address)
  if (listen) {
    await Prepayment.once('SubscriptionConsumerAdded', async (subId, consumer) => {
      console.log('SubscriptionConsumerAdded')
      console.log(`subId ${subId}`)
      console.log(`consumer ${consumer}`)
    })
  }
  await Prepayment.deposit(subId, {value:parseEther(10)})
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
