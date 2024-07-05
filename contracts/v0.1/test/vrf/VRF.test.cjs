// https://github.com/witnet/vrf-solidity/blob/master/test/vrf.js

const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const data = require('./VRF-test-data/data.json')

async function deploy() {
  let { account0: deployer } = await hre.getNamedAccounts()
  deployer = await ethers.getSigner(deployer)
  let contract = await ethers.getContractFactory('TestHelperVRF', {
    signer: deployer.address,
  })
  contract = await contract.deploy()
  await contract.deployed()
  return contract
}

describe('Auxiliary functions: ', () => {
  for (const [index, test] of data.proofs.valid.entries()) {
    it(`should decode a VRF proof from bytes (${index + 1})`, async () => {
      const helper = await loadFixture(deploy)
      const decodedProof = await helper.decodeProof(test.pi)
      expect(decodedProof[0]).to.be.equal(test.gamma.x)
      expect(decodedProof[1]).to.be.equal(test.gamma.y)
      expect(decodedProof[2]).to.be.equal(test.c)
      expect(decodedProof[3]).to.be.equal(test.s)
    })
  }

  for (const [, test] of data.proofs.invalid.entries()) {
    it(`should fail to decode a VRF proof from bytes if malformed (${test.description})`, async () => {
      const helper = await loadFixture(deploy)
      await expect(helper.decodeProof(test.pi)).to.be.rejectedWith(test.revert)
    })
  }

  for (const [index, test] of data.points.valid.entries()) {
    it(`should decode a compressed EC Point (${index + 1})`, async () => {
      const helper = await loadFixture(deploy)
      const coord = await helper.decodePoint(test.compressed)
      expect(coord[0]).to.be.equal(test.uncompressed.x)
      expect(coord[1]).to.be.equal(test.uncompressed.y)
    })
  }

  for (const [, test] of data.points.invalid.entries()) {
    it(`should fail to decode a compressed EC Point if malformed (${test.description})`, async () => {
      const helper = await loadFixture(deploy)
      await expect(helper.decodePoint(test.compressed)).to.be.rejectedWith(test.revert)
    })
  }

  for (const [index, test] of data.computeFastVerifyParams.valid.entries()) {
    it(`should compute fast verify parameters (${index + 1})`, async () => {
      const helper = await loadFixture(deploy)
      const publicKeyX = test.publicKey.x
      const publicKeyY = test.publicKey.y
      const publicKey = [publicKeyX, publicKeyY]
      const proof = await helper.decodeProof(test.pi)
      const message = test.message
      const params = await helper.computeFastVerifyParams(publicKey, proof, message)

      expect(params[0][0]).to.be.equal(test.uPoint.x)
      expect(params[0][1]).to.be.equal(test.uPoint.y)
      expect(params[1][0]).to.be.equal(test.vComponents.sH.x)
      expect(params[1][1]).to.be.equal(test.vComponents.sH.y)
      expect(params[1][2]).to.be.equal(test.vComponents.cGamma.x)
      expect(params[1][3]).to.be.equal(test.vComponents.cGamma.y)
    })
  }

  for (const [, test] of data.computeFastVerifyParams.invalid.entries()) {
    it(`should fail to compute fast verify parameters (${test.description})`, async () => {
      const helper = await loadFixture(deploy)
      const publicKeyX = ethers.BigNumber.from(test.publicKey.x)
      const publicKeyY = test.publicKey.y
      const publicKey = [publicKeyX, publicKeyY]
      const proof = await helper.decodeProof(test.pi)
      const message = test.message
      const params = await helper.computeFastVerifyParams(publicKey, proof, message)

      const results = [
        params[0][0].eq(test.uPoint.x),
        params[0][1].eq(test.uPoint.y),
        params[1][0].eq(test.vComponents.sH.x),
        params[1][1].eq(test.vComponents.sH.y),
        params[1][2].eq(test.vComponents.cGamma.x),
        params[1][3].eq(test.vComponents.cGamma.y),
      ]

      expect(
        results.length === test.asserts.length &&
          results.every((value, index) => value === test.asserts[index]),
      ).to.be.equal(true)
    })
  }
})

describe('Proof verification functions: ', () => {
  for (const [index, test] of data.verify.valid.entries()) {
    it(`should verify a VRF proof (${index + 1})`, async () => {
      const helper = await loadFixture(deploy)
      const publicKey = await helper.decodePoint(test.pub)
      const proof = await helper.decodeProof(test.pi)
      const message = test.message
      const result = await helper.verify(publicKey, proof, message)
      expect(result).to.be.equal(true)
    })
  }

  for (const [, test] of data.verify.invalid.entries()) {
    it(`should return false when verifying an invalid VRF proof (${test.description})`, async () => {
      const helper = await loadFixture(deploy)
      const publicKeyX = test.publicKey.x
      const publicKeyY = test.publicKey.y
      const publicKey = [publicKeyX, publicKeyY]
      const proof = await helper.decodeProof(test.pi)
      const result = await helper.verify(publicKey, proof, test.message)
      expect(result).to.be.equal(false)
    })
  }

  for (const [index, test] of data.fastVerify.valid.entries()) {
    it(`should fast verify a VRF proof (${index + 1})`, async () => {
      const helper = await loadFixture(deploy)
      // Standard inputs
      const proof = await helper.decodeProof(test.pi)
      const publicKeyX = test.publicKey.x
      const publicKeyY = test.publicKey.y
      const publicKey = [publicKeyX, publicKeyY]
      const message = test.message
      // VRF fast verify requirements
      // U = s*B - c*Y
      const uPointX = test.uPoint.x
      const uPointY = test.uPoint.y
      // V = s*H - c*Gamma
      // s*H
      const vProof1X = test.vComponents.sH.x
      const vProof1Y = test.vComponents.sH.y
      // c*Gamma
      const vProof2X = test.vComponents.cGamma.x
      const vProof2Y = test.vComponents.cGamma.y
      // Check
      const result = await helper.fastVerify(
        publicKey,
        proof,
        message,
        [uPointX, uPointY],
        [vProof1X, vProof1Y, vProof2X, vProof2Y],
      )
      expect(result).to.be.equal(true)
    })
  }

  for (const [, test] of data.fastVerify.invalid.entries()) {
    it(`should return false when fast verifying an invalid VRF proof (${test.description})`, async () => {
      const helper = await loadFixture(deploy)
      // Standard inputs
      const proof = await helper.decodeProof(test.pi)
      const publicKeyX = test.publicKey.x
      const publicKeyY = test.publicKey.y
      const publicKey = [publicKeyX, publicKeyY]
      const message = test.message
      // VRF fast verify requirements
      // U = s*B - c*Y
      const uPointX = test.uPoint.x
      const uPointY = test.uPoint.y
      // V = s*H - c*Gamma
      // s*H
      const vProof1X = test.vComponents.sH.x
      const vProof1Y = test.vComponents.sH.y
      // c*Gamma
      const vProof2X = test.vComponents.cGamma.x
      const vProof2Y = test.vComponents.cGamma.y
      // Check
      const result = await helper.fastVerify(
        publicKey,
        proof,
        message,
        [uPointX, uPointY],
        [vProof1X, vProof1Y, vProof2X, vProof2Y],
      )
      expect(result).to.be.equal(false)
    })
  }
})

describe('VRF hash output function: ', () => {
  for (const [index, test] of data.verify.valid.entries()) {
    it(`should generate hash output from VRF proof (${index + 1})`, async () => {
      const helper = await loadFixture(deploy)
      const proof = await helper.decodeProof(test.pi)
      const hash = await helper.gammaToHash(proof[0], proof[1])
      expect(hash).to.be.equal(test.hash)
    })
  }
})
