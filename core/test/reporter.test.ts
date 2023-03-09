import { describe, test, expect } from '@jest/globals'
import { buildWallet, sendTransaction } from '../src/reporter/utils'
import { ethers } from 'ethers'
import { OraklErrorCode } from '../src/errors'

// The following tests have to be run with hardhat network launched.
// If the hardhat cannot be detected tests are skipped.
// TODO Include hardhat network launch to Github Actions pipeline.
describe('Reporter', function () {
  const PROVIDER_URL = 'http://127.0.0.1:8545'
  const PRIVATE_KEY = '0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a' // hardhat account 2

  test('Send payload to invalid address', async function () {
    try {
      const wallet = await buildWallet({
        privateKey: PRIVATE_KEY,
        providerUrl: PROVIDER_URL,
        testConnection: true
      })

      wallet.getBalance()
      const to = '0x000000000000000000000000000000000000000' // wrong address
      const payload = '0x'

      expect(async () => {
        await sendTransaction({ wallet, to, payload })
      }).rejects.toThrow('TxInvalidAddress')
    } catch (e) {
      if (e.code == OraklErrorCode.ProviderNetworkError) {
        return 0
      } else {
        throw e
      }
    }
  })

  test('Send value without insufficient balance', async function () {
    try {
      const privateKey = '0xa5061ebc3567c2d3422807986c1c27425455fa62f4d9286c66d07a9afc6d9869' // account with 0 balance

      const wallet = await buildWallet({
        privateKey,
        providerUrl: PROVIDER_URL,
        testConnection: true
      })

      const to = '0x976EA74026E726554dB657fA54763abd0C3a0aa9' // hardhat account 6
      const value = ethers.utils.parseUnits('1')

      expect(async () => {
        await sendTransaction({ wallet, to, value })
      }).rejects.toThrow('TxProcessingResponseError')
    } catch (e) {
      if (e.code == OraklErrorCode.ProviderNetworkError) {
        return 0
      } else {
        throw e
      }
    }
  })
})
