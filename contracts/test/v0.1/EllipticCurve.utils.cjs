async function deployTestEllipticCurve() {
  let { deployer } = await hre.getNamedAccounts()
  deployer = await ethers.getSigner(deployer)

  let contract = await ethers.getContractFactory('TestEllipticCurve', {
    signer: deployer.address
  })
  contract = await contract.deploy()
  await contract.deployed()

  return contract
}

module.exports = {
  deployTestEllipticCurve
}
