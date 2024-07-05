const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const { parseKlay, createSigners } = require('../utils.cjs')
const { deploy: deployVrfCoordinator } = require('../vrf/VRFCoordinator.utils.cjs')
const { deploy: deployPrepayment, addCoordinator } = require('./Prepayment.utils.cjs')

const {
  deploy: deployRegistry,
  propose,
  confirm,
  createAccount,
  addConsumer,
} = require('./Registry.utils.cjs')

const { deploy: deployCoordinator } = require('./RequestResponseCoordinator.utils.cjs')

async function deploy() {
  const {
    account0: deployerSigner,
    account2,
    account3,
    account4: protocolFeeRecipient,
  } = await createSigners()

  // Prepayment
  const prepaymentContract = await deployPrepayment(protocolFeeRecipient.address, deployerSigner)
  const prepayment = {
    contract: prepaymentContract,
    signer: deployerSigner,
  }

  // VRFCoordinator

  const coordinatorContract = await deployVrfCoordinator(prepaymentContract.address, deployerSigner)
  expect(await coordinatorContract.typeAndVersion()).to.be.equal('VRFCoordinator v0.1')
  const coordinator = {
    contract: coordinatorContract,
    signer: deployerSigner,
  }
  await addCoordinator(prepayment.contract, prepayment.signer, coordinator.contract.address)

  const rRCoordinatorContract = await deployCoordinator(prepayment.contract.address, deployerSigner)
  const rRCoordinator = { contract: rRCoordinatorContract, signer: deployerSigner }
  await addCoordinator(prepayment.contract, prepayment.signer, rRCoordinator.contract.address)

  // registry

  let registryContract = await ethers.getContractFactory('Registry', {
    signer: deployerSigner,
  })
  registryContract = await registryContract.deploy()
  await registryContract.deployed()
  //setup registry

  const fee = parseKlay(1)
  const pChainID = '100001'
  const jsonRpc = 'https://123'
  const L2Endpoint = account2.address
  const { chainID } = await propose(
    registryContract,
    deployerSigner,
    pChainID,
    jsonRpc,
    L2Endpoint,
    fee,
  )
  await confirm(registryContract, deployerSigner, chainID)
  const { accId: rAccId } = await createAccount(registryContract, deployerSigner, chainID)
  //add consumer
  await addConsumer(registryContract, deployerSigner, rAccId, deployerSigner.address)

  let endpointContract = await ethers.getContractFactory('L1Endpoint', {
    signer: deployerSigner,
  })
  endpointContract = await endpointContract.deploy(
    registryContract.address,
    coordinatorContract.address,
    rRCoordinatorContract.address,
  )
  await endpointContract.deployed()
  await endpointContract.addOracle(deployerSigner.address)

  //add endpoint for registry
  await registryContract.setL1Endpoint(endpointContract.address)

  const endpoint = {
    contract: endpointContract,
    signer: deployerSigner,
  }

  const registry = {
    contract: registryContract,
    signer: deployerSigner,
  }

  return {
    prepayment,
    coordinator,
    rRCoordinator,
    endpoint,
    registry,
    account2,
    account3,
    registrAccount: rAccId,
  }
}

describe('L1Endpoint', function () {
  it('add and remove oracle', async function () {
    const { endpoint, account2: oracle } = await loadFixture(deploy)

    const txAdd = await (await endpoint.contract.addOracle(oracle.address)).wait()
    expect(txAdd.events.length).to.be.equal(1)
    const eventAdd = endpoint.contract.interface.parseLog(txAdd.events[0])
    expect(eventAdd.name).to.be.equal('OracleAdded')

    const txRemove = await (await endpoint.contract.removeOracle(oracle.address)).wait()
    expect(txRemove.events.length).to.be.equal(1)
    const eventRemove = endpoint.contract.interface.parseLog(txRemove.events[0])
    expect(eventRemove.name).to.be.equal('OracleRemoved')
  })
})
