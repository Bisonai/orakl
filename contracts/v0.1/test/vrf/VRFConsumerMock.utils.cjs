async function deploy(coordinatorAddress, signer) {
  let contract = await ethers.getContractFactory('VRFConsumerMock', {
    signer,
  })
  contract = await contract.deploy(coordinatorAddress)
  await contract.deployed()
  return contract
}

module.exports = {
  deploy,
}
