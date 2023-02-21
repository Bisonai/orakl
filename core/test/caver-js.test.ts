import { describe, test, expect, jest } from '@jest/globals'
import { BigNumber, ethers } from 'ethers'
import Caver from 'caver-js'

describe('Test Caver-js', function () {
  jest.setTimeout(10000)

  if (process.env.GITHUB_ACTIONS) {
    test('Send signed tx with is caver-js on Baobab', async function () {
      const PROVIDER_URL = 'https://api.baobab.klaytn.net:8651'
      const caver = new Caver(PROVIDER_URL)
      const privateKey = process.env.CAVER_PRIVATE_KEY || ''
      const account1 = caver.klay.accounts.privateKeyToAccount(privateKey)

      const amount = ethers.utils.parseEther('0.001')
      const to = '0xeF5cd886C7f8d85fbe8023291761341aCBb4DA01'
      const beforeBalanceOfTo = await caver.klay.getBalance(to)
      const beforeBalanceOfAccount1 = await caver.klay.getBalance(account1.address)
      const tx = {
        from: account1.address,
        to: to,
        value: amount,
        gas: '21000'
      }
      // Sign transaction
      const signTx = await account1.signTransaction(tx)
      // Send signed transaction
      const txReceipt = await caver.klay.sendSignedTransaction(signTx)

      const txFee = BigNumber.from(txReceipt.effectiveGasPrice).mul(
        BigNumber.from(txReceipt.gasUsed)
      )
      const afterBalanceOfTo = await caver.klay.getBalance(to)
      const afterBalanceOfAccount1 = await caver.klay.getBalance(account1.address)
      expect(
        BigNumber.from(afterBalanceOfTo).eq(
          BigNumber.from(beforeBalanceOfTo).add(BigNumber.from(amount))
        )
      ).toBe(true)
      expect(
        BigNumber.from(afterBalanceOfAccount1).eq(
          BigNumber.from(beforeBalanceOfAccount1).sub(BigNumber.from(amount)).sub(txFee)
        )
      ).toBe(true)
    })
  } else {
    test('Send signed tx with is ethers on local', async function () {
      const provider = new ethers.providers.JsonRpcProvider('http://127.0.0.1:8545')
      const privateKey = '0x4bbbf85ce3377467afe5d46f804f221813b2bb87f24d81f60f1fcdbf7cbf4356' // Hardhat account
      const wallet = await new ethers.Wallet(privateKey, provider)
      const amount = ethers.utils.parseEther('0.001')
      const to = '0xeF5cd886C7f8d85fbe8023291761341aCBb4DA01'
      const beforeBalanceOfTo = await provider.getBalance(to)
      const beforeBalanceOfAccount1 = await provider.getBalance(wallet.address)

      const tx = {
        from: wallet.address,
        to: to,
        value: amount
      }

      // Send transaction
      const txReceipt = await (await wallet.sendTransaction(tx)).wait()
      const afterBalanceOfTo = await provider.getBalance(to)
      const afterBalanceOfAccount1 = await provider.getBalance(wallet.address)
      const txFee = txReceipt.cumulativeGasUsed.mul(txReceipt.effectiveGasPrice)

      expect(afterBalanceOfTo.eq(beforeBalanceOfTo.add(amount))).toBe(true)
      expect(afterBalanceOfAccount1.eq(beforeBalanceOfAccount1.sub(amount).sub(txFee))).toBe(true)
    })
  }
})
