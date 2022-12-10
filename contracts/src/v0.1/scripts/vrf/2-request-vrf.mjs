import pkg from 'hardhat'
const { ethers } = pkg

async function main() {
  const listen = false
  const VRFCoordinatorAddr = '0x5FbDB2315678afecb367f032d93F642f64180aa3'
  const VRFConsumerMockAddr = '0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9'

  let VRFConsumerMock = await ethers.getContractFactory('VRFConsumerMock')
  VRFConsumerMock = await VRFConsumerMock.attach(VRFConsumerMockAddr)
  console.log('VRFConsumerMock Address:', VRFConsumerMock.address)

  let VRFCoordinator = await ethers.getContractFactory('VRFCoordinator')
  VRFCoordinator = await VRFCoordinator.attach(VRFCoordinatorAddr)
  console.log('VRFCoordinator Address:', VRFCoordinator.address)

  await VRFConsumerMock.requestRandomWords()

  if (listen) {
    VRFCoordinator.once(
      'RandomWordsRequested',
      async (
        keyHash,
        requestId,
        preSeed,
        subId,
        requestConfirmations,
        callbackGasLimit,
        numWords,
        sender
      ) => {
        console.log('RandomWordsRequested')
        console.log(`keyHash ${keyHash}`)
        console.log(`requestId ${requestId}`)
        console.log(`preSeed ${preSeed}`)
        console.log(`subId ${subId}`)
        console.log(`requestConfirmations ${requestConfirmations}`)
        console.log(`callbackGasLimit ${callbackGasLimit}`)
        console.log(`numWords ${numWords}`)
        console.log(`sender ${sender}`)
      }
    )
  }
}

main()
