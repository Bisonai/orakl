const { expect } = require('chai')

async function deploy(signer) {
  let contract = await ethers.getContractFactory('Registry', {
    signer
  })
  contract = await contract.deploy()
  await contract.deployed()
  return contract
}

async function propose(
  registry,
  signer,
  pChainID,
  jsonRpc,
  endpoint,
  startRound,
  l1Aggregator,
  l2Aggregator,
  value
) {
  const tx = await (
    await registry
      .connect(signer)
      .proposeChain(pChainID, jsonRpc, endpoint, startRound, l1Aggregator, l2Aggregator, {
        value
      })
  ).wait()
  expect(tx.events.length).to.be.equal(1)
  const event = registry.interface.parseLog(tx.events[0])
  expect(event.name).to.be.equal('ChainProposed')
  const { owner, chainID } = event.args
  return { owner, chainID }
}

async function confirm(registry, signer, pChainID) {
  const tx = await (await registry.connect(signer).confirmChain(pChainID)).wait()
  expect(tx.events.length).to.be.equal(1)
  const event = registry.interface.parseLog(tx.events[0])
  expect(event.name).to.be.equal('ChainConfirmed')
  const { chainID } = event.args
  return { chainID }
}

async function setProposeFee(registry, signer, pFee) {
  const tx = await (await registry.connect(signer).setProposeFee(pFee)).wait()
  expect(tx.events.length).to.be.equal(1)
  const event = registry.interface.parseLog(tx.events[0])
  expect(event.name).to.be.equal('ProposeFeeSet')
  const { fee } = event.args
  return fee
}

async function withdraw(registry, signer, amount) {
  const tx = await (await registry.connect(signer).withdraw(amount)).wait()
}

module.exports = {
  deploy,
  propose,
  confirm,
  setProposeFee,
  withdraw
}
