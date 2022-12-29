import pkg from 'hardhat'
const { ethers } = pkg

async function main() {
  const listen = false
  const VRFConsumerMockAddr = '0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9';


  let VRFConsumerMock = await ethers.getContractFactory('VRFConsumerMock')
  VRFConsumerMock = await VRFConsumerMock.attach(VRFConsumerMockAddr)
  console.log('VRFConsumerMock Address:', VRFConsumerMock.address)
  const randomNumber = await VRFConsumerMock.s_randomResult()
  console.log('randomNumber', randomNumber.toString())

  const PrepaymentAdd = '0x5FbDB2315678afecb367f032d93F642f64180aa3';
  let Prepayment = await ethers.getContractFactory('Prepayment')
  Prepayment = await Prepayment.attach(PrepaymentAdd)
  console.log('Prepayment Address:', Prepayment.address);

  const balance= (await Prepayment.getSubscription(1));
  console.log('sub balance', balance);
  const withdrawer_balance_before=await ethers.provider.getBalance((await Prepayment.owner()));
  console.log('withdraw balance before', withdrawer_balance_before);
  await Prepayment.ownerWithdraw();

  const withdrawer_balance_after=await ethers.provider.getBalance((await Prepayment.owner()));
  console.log('withdraw balance after', withdrawer_balance_after);

}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
