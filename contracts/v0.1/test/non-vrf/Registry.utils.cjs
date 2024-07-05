const { expect } = require('chai')

async function deploy(signer) {
  let contract = await ethers.getContractFactory('Registry', {
    signer,
  })
  contract = await contract.deploy()
  await contract.deployed()
  return contract
}

async function propose(registry, signer, pChainID, jsonRpc, endpoint, value) {
  const tx = await (
    await registry.connect(signer).proposeNewChain(pChainID, jsonRpc, endpoint, {
      value,
    })
  ).wait()
  expect(tx.events.length).to.be.equal(1)
  const event = registry.interface.parseLog(tx.events[0])
  expect(event.name).to.be.equal('ChainProposed')
  const { owner, chainID } = event.args
  return { owner, chainID }
}

async function editChainInfor(registry, signer, pChainID, jsonRpc, pEndpoint, value) {
  const tx = await (
    await registry.connect(signer).editChainInfo(pChainID, jsonRpc, pEndpoint, {
      value,
    })
  ).wait()
  expect(tx.events.length).to.be.equal(1)
  const event = registry.interface.parseLog(tx.events[0])
  expect(event.name).to.be.equal('ChainEdited')
  const { rpc, endpoint } = event.args
  return { rpc, endpoint }
}

async function confirm(registry, signer, pChainID) {
  const tx = await (await registry.connect(signer).confirmChain(pChainID)).wait()
  expect(tx.events.length).to.be.equal(1)
  const event = registry.interface.parseLog(tx.events[0])
  expect(event.name).to.be.equal('ChainConfirmed')
  const { chainID } = event.args
  return { chainID }
}

async function addAggregator(registry, signer, pChainID, l1Aggregator, l2Aggregator) {
  const tx = await (
    await registry.connect(signer).addAggregator(pChainID, l1Aggregator, l2Aggregator)
  ).wait()
  expect(tx.events.length).to.be.equal(1)
  const event = registry.interface.parseLog(tx.events[0])
  expect(event.name).to.be.equal('AggregatorAdded')
  const { chainID, aggregatorID } = event.args
  return { chainID, aggregatorID }
}

async function removeAggregator(registry, signer, pChainID, aggregatorId) {
  const tx = await (await registry.connect(signer).removeAggregator(pChainID, aggregatorId)).wait()
  expect(tx.events.length).to.be.equal(1)
  const event = registry.interface.parseLog(tx.events[0])
  expect(event.name).to.be.equal('AggregatorRemoved')
  const { chainID, aggregatorID } = event.args
  return { chainID, aggregatorID }
}

async function createAccount(registry, signer, pChainID) {
  const tx = await (await registry.connect(signer).createAccount(pChainID)).wait()
  expect(tx.events.length).to.be.equal(1)
  const event = registry.interface.parseLog(tx.events[0])
  expect(event.name).to.be.equal('AccountCreated')
  const { accId, chainId, owner } = event.args
  return { accId, chainId, owner }
}

async function addConsumer(registry, signer, accId, consumerAddress) {
  const tx = await (await registry.connect(signer).addConsumer(accId, consumerAddress)).wait()
  expect(tx.events.length).to.be.equal(1)
  const event = registry.interface.parseLog(tx.events[0])
  expect(event.name).to.be.equal('ConsumerAdded')
}

async function removeConsumer(registry, signer, accId, consumerAddress) {
  const tx = await (await registry.connect(signer).removeConsumer(accId, consumerAddress)).wait()
  expect(tx.events.length).to.be.equal(1)
  const event = registry.interface.parseLog(tx.events[0])
  expect(event.name).to.be.equal('ConsumerRemoved')
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
  withdraw,
  editChainInfor,
  createAccount,
  addAggregator,
  removeAggregator,
  addConsumer,
  removeConsumer,
}
