import { beforeEach, describe, expect, test } from '@jest/globals'
import { ethers } from 'ethers'
import pino, { Logger } from 'pino'
import { OraklErrorCode } from '../src/errors'
import {
  buildCaverWallet,
  buildWallet,
  sendTransaction,
  sendTransactionDelegatedFee
} from '../src/reporter/utils'

// The following tests have to be run with hardhat network launched.
// If the hardhat cannot be detected tests are skipped.
describe('Reporter', function () {
  const PROVIDER_URL = 'http://127.0.0.1:8545'
  const PRIVATE_KEY = '0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a' // hardhat account 2
  let logger: Logger

  beforeEach(() => {
    const transport = pino.transport({
      target: 'pino/file',
      options: { destination: '/dev/null' }
    })
    logger = pino(transport)
  })

  // Test only for local network. Test must be running, otherwise test fail!
  if (!process.env.GITHUB_ACTIONS && PROVIDER_URL == 'http://127.0.0.1:8545') {
    test('Send payload to invalid address', async function () {
      try {
        const wallet = await buildWallet({
          privateKey: PRIVATE_KEY,
          providerUrl: PROVIDER_URL
        })

        const to = '0x000000000000000000000000000000000000000' // wrong address
        const payload = '0x'

        expect(async () => {
          await sendTransaction({ wallet, to, payload, logger })
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
          providerUrl: PROVIDER_URL
        })

        const to = '0x976EA74026E726554dB657fA54763abd0C3a0aa9' // hardhat account 6
        const value = ethers.utils.parseUnits('1')

        expect(async () => {
          await sendTransaction({ wallet, to, value, logger })
        }).rejects.toThrow('TxInsufficientFunds')
      } catch (e) {
        if (e.code == OraklErrorCode.ProviderNetworkError) {
          return 0
        } else {
          throw e
        }
      }
    })
  } else {
    // TODO Include hardhat network launch to Github Actions pipeline.
    test('Dummy test', function () {
      expect(true).toBe(true)
    })
  }

  // TODO set up for CI/CD with Orakl Network Delegator running in Bobab
  test.skip('Test Delegated Transaction Sign', async function () {
    const COUNTER = {
      // Baobab
      address: '0x26532aabc377ee02a8b35ff770ef5660881787db',
      abi: [
        {
          inputs: [],
          name: 'increment',
          outputs: [],
          stateMutability: 'nonpayable',
          type: 'function'
        }
      ]
    }

    const wallet = buildCaverWallet({
      // 0 $KLAY in Account
      // address: '0x9bf123A486DD67d5B2B859c74BFa3035c99b9243'
      privateKey: '0xaa8707622845b72c76b7b9f329b154140441eda385ca39e3cdc66d2bee5f98e0',
      providerUrl: 'https://api.baobab.klaytn.net:8651'
    })

    const iface = new ethers.utils.Interface(COUNTER.abi)
    const payload = iface.encodeFunctionData('increment')

    await sendTransactionDelegatedFee({
      wallet,
      to: COUNTER.address,
      payload,
      logger,
      gasLimit: 100_000
    })
  })
})
