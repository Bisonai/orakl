import pkg from 'hardhat'
const { ethers } = pkg

async function main() {
  const listen = false
  const VRFConsumerMockAddr = '0x9A676e781A523b5d0C0e43731313A708CB607508';


  let VRFConsumerMock = await ethers.getContractFactory('VRFConsumerMock')
  VRFConsumerMock = await VRFConsumerMock.attach(VRFConsumerMockAddr)
  console.log('VRFConsumerMock Address:', VRFConsumerMock.address)
  const randomNumber = await VRFConsumerMock.s_randomResult()
  console.log('randomNumber', randomNumber.toString())


  const PrepaymentAdd = '0x610178dA211FEF7D417bC0e6FeD39F05609AD788';
  let Prepayment = await ethers.getContractFactory('Prepayment')
  Prepayment = await Prepayment.attach(PrepaymentAdd)
  console.log('Prepayment Address:', Prepayment.address);

  const balance= (await Prepayment.getSubscription(1));
    console.log('balance', balance);

}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
