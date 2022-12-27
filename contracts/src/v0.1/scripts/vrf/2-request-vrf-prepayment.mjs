import pkg from 'hardhat'
const { ethers } = pkg

async function main() {
  const listen = true
  const VRFCoordinatorAddr = '0xB7f8BC63BbcaD18155201308C8f3540b07f84F5e'
  const VRFConsumerMockAddr = '0x9A676e781A523b5d0C0e43731313A708CB607508'

  let VRFConsumerMock = await ethers.getContractFactory('VRFConsumerMock')
  VRFConsumerMock = await VRFConsumerMock.attach(VRFConsumerMockAddr)
  console.log('VRFConsumerMock Address:', VRFConsumerMock.address)

  let VRFCoordinator = await ethers.getContractFactory('VRFCoordinator')
  VRFCoordinator = await VRFCoordinator.attach(VRFCoordinatorAddr)
  console.log('VRFCoordinator Address:', VRFCoordinator.address)

  await VRFConsumerMock.requestRandomWords()

  if (listen || true) {
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

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
