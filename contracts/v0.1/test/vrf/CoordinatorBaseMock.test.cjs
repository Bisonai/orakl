const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const { parseKlay, createSigners } = require('../utils.cjs')

const MAX_GAS_LIMIT = 0 // not testing for this parameter
const GAS_AFTER_PAYMENT_CALCULATION = 0 // not testing for this parameter

async function deploy() {
  const { account0: deployerSigner } = await createSigners()

  // CoordinatorBaseMock
  let contract = await ethers.getContractFactory('CoordinatorBaseMock', {
    signer: deployerSigner,
  })
  contract = await contract.deploy()
  await contract.deployed()

  return {
    contract,
  }
}

describe('CoordinatorBaseMock', function () {
  it('Test tier without initialization', async function () {
    const { contract } = await loadFixture(deploy)
    expect(await contract.computeFee(1)).to.be.equal(0)
  })

  it('Test tier fee w/o range calculation correctness', async function () {
    const { contract } = await loadFixture(deploy)

    // Fee configuration with price 1 $KLAY for any request
    const feeConfig = {
      fulfillmentFlatFeeKlayPPMTier1: 1_000_000,
      fulfillmentFlatFeeKlayPPMTier2: 1_000_000,
      fulfillmentFlatFeeKlayPPMTier3: 1_000_000,
      fulfillmentFlatFeeKlayPPMTier4: 1_000_000,
      fulfillmentFlatFeeKlayPPMTier5: 1_000_000,
      reqsForTier2: 0,
      reqsForTier3: 0,
      reqsForTier4: 0,
      reqsForTier5: 0,
    }
    await contract.setConfig(MAX_GAS_LIMIT, GAS_AFTER_PAYMENT_CALCULATION, Object.values(feeConfig))

    expect(parseKlay('1')).to.be.equal(await contract.computeFee(1))
    expect(parseKlay('1')).to.be.equal(await contract.computeFee(1_000))
    expect(parseKlay('1')).to.be.equal(await contract.computeFee(1_000_000))
  })

  it('Test tier fee w/ range calculation correctness', async function () {
    const { contract } = await loadFixture(deploy)
    // Fee configuration with decreasing price of 1 $KLAY for every higher tier
    const feeConfig = {
      fulfillmentFlatFeeKlayPPMTier1: 5_000_000,
      fulfillmentFlatFeeKlayPPMTier2: 4_000_000,
      fulfillmentFlatFeeKlayPPMTier3: 3_000_000,
      fulfillmentFlatFeeKlayPPMTier4: 2_000_000,
      fulfillmentFlatFeeKlayPPMTier5: 1_000_000,
      reqsForTier2: 10,
      reqsForTier3: 20,
      reqsForTier4: 30,
      reqsForTier5: 40,
    }
    await contract.setConfig(MAX_GAS_LIMIT, GAS_AFTER_PAYMENT_CALCULATION, Object.values(feeConfig))

    // First tier
    // 0 <= reqCount && reqCount <= fc.reqsForTier2
    expect(parseKlay('5')).to.be.equal(await contract.computeFee(0))
    expect(parseKlay('5')).to.be.equal(await contract.computeFee(feeConfig.reqsForTier2))

    // Second tier
    // fc.reqsForTier2 < reqCount && reqCount <= fc.reqsForTier3
    expect(parseKlay('4')).to.be.equal(await contract.computeFee(feeConfig.reqsForTier2 + 1))
    expect(parseKlay('4')).to.be.equal(await contract.computeFee(feeConfig.reqsForTier3))

    // Third tier
    // fc.reqsForTier3 < reqCount && reqCount <= fc.reqsForTier4
    expect(parseKlay('3')).to.be.equal(await contract.computeFee(feeConfig.reqsForTier3 + 1))
    expect(parseKlay('3')).to.be.equal(await contract.computeFee(feeConfig.reqsForTier4))

    // Fourth tier
    // fc.reqsForTier4 < reqCount && reqCount <= fc.reqsForTier5
    expect(parseKlay('2')).to.be.equal(await contract.computeFee(feeConfig.reqsForTier4 + 1))
    expect(parseKlay('2')).to.be.equal(await contract.computeFee(feeConfig.reqsForTier5))

    // Fifth tier
    // reqCount > fc.reqsForTier5
    expect(parseKlay('1')).to.be.equal(await contract.computeFee(feeConfig.reqsForTier5 + 1))
    expect(parseKlay('1')).to.be.equal(
      await contract.computeFee(feeConfig.reqsForTier4 + 1_000_000),
    )
  })
})
