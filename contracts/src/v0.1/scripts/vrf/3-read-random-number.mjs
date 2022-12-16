import pkg from 'hardhat'
const { ethers } = pkg

async function main() {
  const listen = false
  const VRFConsumerMockAddr = '0x5F1f5dE8EAA2CebC554F3DA50Fe9a9BA5Cf300ff'

  let VRFConsumerMock = await ethers.getContractFactory('VRFConsumerMock')
  VRFConsumerMock = await VRFConsumerMock.attach(VRFConsumerMockAddr)
  console.log('VRFConsumerMock Address:', VRFConsumerMock.address)
  const randomNumber = await VRFConsumerMock.s_randomResult()
  console.log('randomNumber', randomNumber.toString())
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
