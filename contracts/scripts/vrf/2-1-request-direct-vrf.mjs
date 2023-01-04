import pkg from 'hardhat'
const { ethers } = pkg

async function main() {
  const listen = false
  const VRFCoordinatorAddr = '0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512'
  const VRFConsumerMockAddr = '0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9'

  let VRFConsumerMock = await ethers.getContractFactory('VRFConsumerMock')
  VRFConsumerMock = await VRFConsumerMock.attach(VRFConsumerMockAddr)
  console.log('VRFConsumerMock Address:', VRFConsumerMock.address)

  let VRFCoordinator = await ethers.getContractFactory('VRFCoordinator')
  VRFCoordinator = await VRFCoordinator.attach(VRFCoordinatorAddr)
  console.log('VRFCoordinator Address:', VRFCoordinator.address)

  await VRFConsumerMock.requestRandomWordsDirect({ value: '100000000000000000' })
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
