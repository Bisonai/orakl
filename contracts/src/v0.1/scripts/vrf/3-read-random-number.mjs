import pkg from 'hardhat'
const { ethers } = pkg

async function main() {
  const listen = false
  const VRFConsumerMockAddr = '0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9'

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
