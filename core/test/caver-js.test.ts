import { describe, test, expect, jest } from '@jest/globals'
import { BigNumber, ethers } from 'ethers'
import Caver from 'caver-js'

const PROVIDER_URL = 'https://api.baobab.klaytn.net:8651'

// if (NODE_ENV != 'development') {
//   PROVIDER_URL = 'https://api.baobab.klaytn.net:8651'
// } else {
//   PROVIDER_URL = 'http://127.0.0.1:8551'
// }

const caver = new Caver(PROVIDER_URL)
const password = 'trb}RROVYs#ye2rq'
const key = {
  address: '14fadded4f98064c3188765df46a055edebf0cf2',
  keyring: [
    [
      {
        cipher: 'aes-128-ctr',
        ciphertext: 'acdce40f546d876430ec072118fe1d71c86e727e9ab03b70581bb6d9e3d7e48a',
        cipherparams: { iv: '59f02c6cf41fc704dd8fe700b763b607' },
        kdf: 'scrypt',
        kdfparams: {
          dklen: 32,
          n: 262144,
          p: 1,
          r: 8,
          salt: '226fa77e4832c3aabbe83e0ed896e08b56880f83e4a593ef5b58ea5d1185e345'
        },
        mac: '3a29ff9234099b04a95b1009d884fe82610ea05e8c745332c550a118e5255b5c'
      }
    ]
  ],
  id: 'fe0ee297-6fdd-414f-991b-e7f563a10b2d',
  version: 4
}

describe('Reporter', function () {
  test('Send signed tx with is caver-js', async function () {
    console.log('Started')
    const account1 = caver.klay.accounts.decrypt(key, password)
    console.log('Account', account1)
    caver.klay.accounts.wallet.add(account1.privateKey)
    console.log('Wallet connected')
    jest.setTimeout(30000)
    const amount = ethers.utils.parseEther('0.001')
    const to = '0xeF5cd886C7f8d85fbe8023291761341aCBb4DA01'
    const beforeBalanceOfTo = await caver.klay.getBalance(to)
    const beforeBalanceOfAccount1 = await caver.klay.getBalance(account1.address)

    console.log(beforeBalanceOfTo)
    console.log(beforeBalanceOfAccount1)

    // const tx = {
    //   from: account1.address,
    //   to: to,
    //   value: amount,
    //   gas: '300000'
    // }

    // // Sign transaction
    // const signTx: any = await caver.klay.accounts.signTransaction(tx)

    // // Send signed transaction
    // const txReceipt = await caver.klay.sendSignedTransaction(signTx)
    // const txFee = BigNumber.from(txReceipt.effectiveGasPrice).mul(BigNumber.from(txReceipt.gasUsed))
    // const afterBalanceOfTo = await caver.klay.getBalance(to)
    // const afterBalanceOfAccount1 = await caver.klay.getBalance(account1.address)

    // expect(
    //   BigNumber.from(afterBalanceOfTo).eq(
    //     BigNumber.from(beforeBalanceOfTo).add(BigNumber.from(amount))
    //   )
    // ).toBe(true)
    // expect(
    //   BigNumber.from(afterBalanceOfAccount1).eq(
    //     BigNumber.from(beforeBalanceOfAccount1).sub(BigNumber.from(amount)).sub(txFee)
    //   )
    // ).toBe(true)
  })
})
