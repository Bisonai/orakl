import pkg from 'hardhat'
const { ethers } = pkg

async function main() {
  const listen = false
  const VRFCoordinatorAddr = '0xC1d68b061DD3E2Fa3eD666FE4075A272a994D783'
  const VRFConsumerMockAddr = '0x5F1f5dE8EAA2CebC554F3DA50Fe9a9BA5Cf300ff'

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
