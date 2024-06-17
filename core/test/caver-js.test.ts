import { describe, expect, jest, test } from '@jest/globals'
import Caver from 'caver-js'
import { BigNumber, ethers } from 'ethers'

describe('Test Caver-js', function () {
  jest.setTimeout(10000)

  if (process.env.GITHUB_ACTIONS) {
    test('Send signed tx with is caver-js on Baobab', async function () {
      const PROVIDER_URL = 'https://public-en.kairos.node.kaia.io'
      const caver = new Caver(PROVIDER_URL)
      const privateKey = process.env.CAVER_PRIVATE_KEY || ''
      const account = caver.klay.accounts.wallet.add(privateKey)

      const amount = caver.utils.toPeb(1, 'mKLAY')
      const to = '0xeF5cd886C7f8d85fbe8023291761341aCBb4DA01'
      const beforeBalanceOfTo = await caver.klay.getBalance(to)
      const beforeBalanceOfAccount = await caver.klay.getBalance(account.address)

      const txReceipt = await caver.klay.sendTransaction({
        type: 'VALUE_TRANSFER',
        from: account.address,
        to: to,
        gas: '21000',
        value: amount
      })

      const txFee = BigNumber.from(txReceipt.effectiveGasPrice).mul(
        BigNumber.from(txReceipt.gasUsed)
      )
      const afterBalanceOfTo = await caver.klay.getBalance(to)
      const afterBalanceOfAccount = await caver.klay.getBalance(account.address)

      expect(
        BigNumber.from(afterBalanceOfTo).eq(
          BigNumber.from(beforeBalanceOfTo).add(BigNumber.from(amount))
        )
      ).toBe(true)
      expect(
        BigNumber.from(afterBalanceOfAccount).eq(
          BigNumber.from(beforeBalanceOfAccount).sub(BigNumber.from(amount)).sub(txFee)
        )
      ).toBe(true)
    }, 60_000)
  } else {
    test('Send signed tx with is ethers on local', async function () {
      const provider = new ethers.providers.JsonRpcProvider('http://127.0.0.1:8545')
      const privateKey = '0x4bbbf85ce3377467afe5d46f804f221813b2bb87f24d81f60f1fcdbf7cbf4356' // Hardhat account
      const wallet = await new ethers.Wallet(privateKey, provider)
      const amount = ethers.utils.parseEther('0.001')
      const to = '0xeF5cd886C7f8d85fbe8023291761341aCBb4DA01'
      const beforeBalanceOfTo = await provider.getBalance(to)
      const beforeBalanceOfAccount = await provider.getBalance(wallet.address)

      const tx = {
        from: wallet.address,
        to: to,
        value: amount
      }

      // Send transaction
      const txReceipt = await (await wallet.sendTransaction(tx)).wait()
      const afterBalanceOfTo = await provider.getBalance(to)
      const afterBalanceOfAccount = await provider.getBalance(wallet.address)
      const txFee = txReceipt.cumulativeGasUsed.mul(txReceipt.effectiveGasPrice)

      expect(afterBalanceOfTo.eq(beforeBalanceOfTo.add(amount))).toBe(true)
      expect(afterBalanceOfAccount.eq(beforeBalanceOfAccount.sub(amount).sub(txFee))).toBe(true)
    })
  }
})
