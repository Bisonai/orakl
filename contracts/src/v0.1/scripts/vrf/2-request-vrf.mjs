import pkg from 'hardhat'
const { ethers } = pkg

async function main() {
  let VRFConsumerMock = await ethers.getContractFactory('VRFConsumerMock')
  VRFConsumerMock = await VRFConsumerMock.attach('0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0')
  console.log('VRFConsumerMock Address:', VRFConsumerMock.address)

  let VRFCoordinator = await ethers.getContractFactory('VRFCoordinator')
  VRFCoordinator = await VRFCoordinator.attach('0x5FbDB2315678afecb367f032d93F642f64180aa3')
  console.log('VRFCoordinator Address:', VRFCoordinator.address)

  await VRFConsumerMock.requestRandomWords()

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

main()
