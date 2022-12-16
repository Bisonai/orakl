/// step 1: deploy VRF Coordinator
import pkg from 'hardhat'
const { ethers } = pkg
//require('dotenv').config()

const ZERO_ADDRESS = ethers.constants.AddressZero
// const for proving key
const VRFCoordinatorAddress = "0x7F690d91028925Bc5324FBcFCA0Da31A6E994bF2"
const VRFConsumerMock = "0xac509ECd241a339a23607a0f7c234AaBf4A9E946"

async function deploy() {
    let VRFCoordinator = await ethers.getContractFactory('VRFCoordinator')
    VRFCoordinator = await VRFCoordinator.deploy()
    await VRFCoordinator.deployed()
    console.log('VRFCoordinator Address:', VRFCoordinator.address)
}
async function registerProvingKey() {
    const publicProvingKey = [
        '95162740466861161360090244754314042169116280320223422208903791243647772670481',
        '53113177277038648369733569993581365384831203706597936686768754351087979105423'
      ]
    let VRFCoordinatorFactory = await ethers.getContractFactory('VRFCoordinator')
    let VRFCoordinator = await VRFCoordinatorFactory.attach(VRFCoordinatorAddress)
    let tx = await VRFCoordinator.registerProvingKey("0x72eBC7770884117fd2b6aA322A977a8Adb0527ee", publicProvingKey)
   
    VRFCoordinator.once('ProvingKeyRegistered', async (keyHash, oracle) => {
        console.log(`keyHash ${keyHash}`)
        console.log(`oracle ${oracle}`)
      })

    console.log(tx)
}
async function setConfig() {
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
  let VRFCoordinatorFactory = await ethers.getContractFactory('VRFCoordinator')
  let VRFCoordinator = await VRFCoordinatorFactory.attach(VRFCoordinatorAddress)
  // Configure VRF Coordinator
   let tx = await VRFCoordinator.setConfig(
    minimumRequestConfirmations,
    maxGasLimit,
    gasAfterPaymentCalculation,
    feeConfig
  )
  console.log(tx)
}
async function deployConsumer() {
  let VRFConsumerMock = await ethers.getContractFactory('VRFConsumerMock')
  VRFConsumerMock = await VRFConsumerMock.deploy(VRFCoordinatorAddress)
  await VRFConsumerMock.deployed()
  console.log('VRFConsumerMock Address:', VRFConsumerMock.address)
}
async function CreateSubscription() {
    let VRFCoordinatorFactory = await ethers.getContractFactory('VRFCoordinator')
    let VRFCoordinator = await VRFCoordinatorFactory.attach(VRFCoordinatorAddress)
    let Tx = await VRFCoordinator.createSubscription()

    console.log(await Tx.wait())
}
async function AddConsumer() {
    let VRFCoordinatorFactory = await ethers.getContractFactory('VRFCoordinator')
    let VRFCoordinator = await VRFCoordinatorFactory.attach(VRFCoordinatorAddress)
    let Tx = await VRFCoordinator.addConsumer(1,VRFConsumerMock)

    console.log(await Tx.wait())
}
async function getRandom() {
    let VRFCoordinatorFactory = await ethers.getContractFactory('VRFCoordinator')
    let VRFCoordinator = await VRFCoordinatorFactory.attach(VRFCoordinatorAddress)

    let Consumer = await ethers.getContractFactory('VRFConsumerMock')
    let consumer = await Consumer.attach(VRFConsumerMock)
    const ID =  await consumer.requestRandomWords()
    console.log(ID)
}
//step 1: deploy VRFCoordinator
//deploy()
//step 2: register proving key
//registerProvingKey()
//step 3: set config
//setConfig()
//step 4: deploy consumer
//deployConsumer()
//step 5: create subscription
//CreateSubscription()
//step 6: add consumer
//AddConsumer()
//step 7: test get random from consumer
getRandom()