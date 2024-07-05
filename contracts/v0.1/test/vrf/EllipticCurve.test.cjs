// https://github.com/witnet/elliptic-curve-solidity/blob/master/test/ellipticCurve.js
const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')

async function deploy() {
  let { account0: deployer } = await hre.getNamedAccounts()
  deployer = await ethers.getSigner(deployer)

  let contract = await ethers.getContractFactory('TestEllipticCurve', {
    signer: deployer.address,
  })
  contract = await contract.deploy()
  await contract.deployed()

  return contract
}

// /////////////////////////////////////////// //
// Check auxiliary operations for given curves //
// /////////////////////////////////////////// //
const auxCurves = ['secp256k1', 'P256']
for (const curve of auxCurves) {
  describe(`Aux. operations - Curve ${curve}`, () => {
    const curveData = require(`./EC-test-data/${curve}-aux.json`)

    const pp = curveData.params.pp
    const aa = curveData.params.aa
    const bb = curveData.params.bb

    // toAffine
    for (const [index, test] of curveData.toAffine.valid.entries()) {
      it(`should convert a Jacobian point to affine (${index + 1})`, async () => {
        const ecLib = await loadFixture(deploy)
        const affine = await ecLib.toAffine(test.input.x, test.input.y, test.input.z, pp)
        const expectedX = test.output.x
        expect(affine[0]).to.be.equal(test.output.x)
        expect(affine[1]).to.be.equal(test.output.y)
      })
    }

    // invMod
    for (const [index, test] of curveData.invMod.valid.entries()) {
      it(`should invert a scalar (${index + 1}) - ${test.description}`, async () => {
        const ecLib = await loadFixture(deploy)
        const inv = await ecLib.invMod(test.input.k, pp)
        expect(inv).to.be.equal(test.output.k)
      })
    }

    // invMod - invalid inputs
    for (const [index, test] of curveData.invMod.invalid.entries()) {
      it(`should fail when inverting with invalid inputs (${index + 1}) - ${
        test.description
      }`, async () => {
        const ecLib = await loadFixture(deploy)
        await expect(ecLib.invMod(test.input.k, test.input.mod)).to.be.rejectedWith(
          test.output.error,
        )
      })
    }

    // expMod
    for (const [index, test] of curveData.expMod.valid.entries()) {
      it(`should do an expMod with ${test.description} - (${index + 1})`, async () => {
        const ecLib = await loadFixture(deploy)
        const exp = await ecLib.expMod(test.input.base, test.input.exp, pp)
        expect(exp).to.be.equal(test.output.k)
      })
    }

    // deriveY
    for (const [index, test] of curveData.deriveY.valid.entries()) {
      it(`should decode coordinate y from compressed point (${index + 1})`, async () => {
        const ecLib = await loadFixture(deploy)
        const coordY = await ecLib.deriveY(test.input.sign, test.input.x, aa, bb, pp)
        expect(coordY).to.be.equal(test.output.y)
      })
    }

    // isOnCurve
    for (const [index, test] of curveData.isOnCurve.valid.entries()) {
      it(`should identify if point is on the curve (${index + 1}) - ${
        test.output.isOnCurve
      }`, async () => {
        const ecLib = await loadFixture(deploy)
        expect(await ecLib.isOnCurve(test.input.x, test.input.y, aa, bb, pp)).to.be.equal(
          test.output.isOnCurve,
        )
      })
    }

    // invertPoint
    for (const [index, test] of curveData.invertPoint.valid.entries()) {
      it(`should invert an EC point (${index + 1})`, async () => {
        const ecLib = await loadFixture(deploy)
        const invertedPoint = await ecLib.ecInv(test.input.x, test.input.y, pp)

        expect(invertedPoint[0]).to.be.equal(test.output.x)
        expect(invertedPoint[1]).to.be.equal(test.output.y)
      })
    }
  })
}

// /////////////////////////////////////////////// //
// Check EC arithmetic operations for given curves //
// /////////////////////////////////////////////// //
const curves = ['secp256k1', 'secp192k1', 'secp224k1', 'P256', 'P192', 'P224']
for (const curve of curves) {
  describe(`Arithmetic operations - Curve ${curve}`, () => {
    const curveData = require(`./EC-test-data/${curve}.json`)

    const pp = curveData.params.pp
    const aa = curveData.params.aa

    // Addition
    for (const [index, test] of curveData.addition.valid.entries()) {
      it(`should add two numbers (${index + 1}) - ${test.description}`, async () => {
        const ecLib = await loadFixture(deploy)
        const res = await ecLib.ecAdd(
          test.input.x1,
          test.input.y1,
          test.input.x2,
          test.input.y2,
          aa,
          pp,
        )

        expect(res[0]).to.be.equal(test.output.x)
        expect(res[1]).to.be.equal(test.output.y)
      })
    }

    // Subtraction
    for (const [index, test] of curveData.subtraction.valid.entries()) {
      it(`should subtract two numbers (${index + 1}) - ${test.description}`, async () => {
        const ecLib = await loadFixture(deploy)
        const res = await ecLib.ecSub(
          test.input.x1,
          test.input.y1,
          test.input.x2,
          test.input.y2,
          aa,
          pp,
        )

        expect(res[0]).to.be.equal(test.output.x)
        expect(res[1]).to.be.equal(test.output.y)
      })
    }

    // Multiplication
    for (const [index, test] of curveData.multiplication.valid.entries()) {
      it(`should multiply EC points (${index + 1}) - ${test.description}`, async () => {
        const ecLib = await loadFixture(deploy)
        const res = await ecLib.ecMul(test.input.k, test.input.x, test.input.y, aa, pp)
        expect(res[0]).to.be.equal(test.output.x)
        expect(res[1]).to.be.equal(test.output.y)
      })
    }
  })
}
