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
  const oracle = '0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199' // Hardhat account 19
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
  const oracleRole=await Prepayment.ORACLE_ROLE();
  await Prepayment.grantRole(oracleRole,VRFCoordinator.address)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
