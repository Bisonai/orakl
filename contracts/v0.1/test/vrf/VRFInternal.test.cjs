// https://github.com/witnet/vrf-solidity/blob/master/test/internal.js

const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const data = require('./VRF-test-data/data.json')

async function deploy() {
  let { account0: deployer } = await hre.getNamedAccounts()
  deployer = await ethers.getSigner(deployer)
  let contract = await ethers.getContractFactory('TestHelperVRFInternals', {
    signer: deployer.address,
  })
  contract = await contract.deploy()
  await contract.deployed()
  return contract
}

describe('VRF underlying algorithms: ', () => {
  for (const [, test] of data.hashToTryAndIncrement.valid.entries()) {
    it(`Hash to Try And Increment (TAI) (${test.description})`, async () => {
      const helper = await loadFixture(deploy)

      const publicKeyX = test.publicKey.x
      const publicKeyY = test.publicKey.y
      const publicKey = [publicKeyX, publicKeyY]
      const message = test.message
      const result = await helper.hashToTryAndIncrement(publicKey, message)

      expect(result[0]).to.be.equal(test.hashPoint.x)
      expect(result[1]).to.be.equal(test.hashPoint.y)
    })
  }

  for (const [index, test] of data.hashPoints.valid.entries()) {
    it(`Points to hash (digest from EC points) (${index + 1})`, async () => {
      const helper = await loadFixture(deploy)
      const res = await helper.hashPoints(
        test.hPoint.x,
        test.hPoint.y,
        test.gamma.x,
        test.gamma.y,
        test.uPoint.x,
        test.uPoint.y,
        test.vPoint.x,
        test.vPoint.y,
      )
      expect(res).to.be.equal(test.hash)
    })
  }
})

describe('VRF internal aux. functions: ', () => {
  for (const [index, point] of data.points.valid.entries()) {
    it(`should encode an EC point to compressed format (${index + 1})`, async () => {
      const helper = await loadFixture(deploy)
      const res = await helper.encodePoint(point.uncompressed.x, point.uncompressed.y)
      expect(res).to.be.equal(point.compressed)
    })
  }

  for (const [index, test] of data.ecMulSubMul.valid.entries()) {
    it(`should do an ecMulSubMul operation (${index + 1})`, async () => {
      const helper = await loadFixture(deploy)
      const res = await helper.ecMulSubMul(
        test.scalar1,
        test.a1,
        test.a2,
        test.scalar2,
        test.b1,
        test.b2,
      )
      expect(res[0]).to.be.equal(test.output.x)
      expect(res[1]).to.be.equal(test.output.y)
    })
  }

  for (const [index, test] of data.ecMul.valid.entries()) {
    it(`should verify an ecMul operation (ecrecover hack) (${index + 1})`, async () => {
      const helper = await loadFixture(deploy)
      const res = await helper.ecMulVerify(
        test.scalar,
        test.x,
        test.y,
        test.output.x,
        test.output.y,
      )
      expect(res).to.be.equal(true)
    })
  }

  for (const [index, test] of data.ecMulSubMulVerify.valid.entries()) {
    it(`should verify an ecMulSubMul operation (ecrecover hack enhanced) (${
      index + 1
    })`, async () => {
      const helper = await loadFixture(deploy)
      const res = await helper.ecMulSubMulVerify(
        test.scalar1,
        test.scalar2,
        test.x,
        test.y,
        test.output.x,
        test.output.y,
      )
      expect(res).to.be.equal(true)
    })
  }
})
