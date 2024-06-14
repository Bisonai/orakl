import { beforeEach, describe, expect, jest, test } from '@jest/globals'
import { ethers } from 'ethers'
import pino, { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { createClient } from 'redis'
import { OraklErrorCode } from '../src/errors'
import { buildMockLogger } from '../src/logger'
import { State } from '../src/reporter/state'
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
        const nonce = await wallet.getTransactionCount()

        const to = '0x000000000000000000000000000000000000000' // wrong address
        const payload = '0x'

        expect(async () => {
          await sendTransaction({ wallet, to, payload, logger, nonce })
        }).rejects.toThrow('TxInvalidAddress')
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
      providerUrl: 'https://public-en.kairos.node.kaia.io'
    })
    const nonce = Number(await wallet.caver.rpc.klay.getTransactionCount(wallet.address))

    const iface = new ethers.utils.Interface(COUNTER.abi)
    const payload = iface.encodeFunctionData('increment')

    await sendTransactionDelegatedFee({
      wallet,
      to: COUNTER.address,
      payload,
      logger,
      gasLimit: 100_000,
      nonce
    })
  })
})

describe('Filter invalid reporters inside of State', function () {
  const PROVIDER_URL = process.env.GITHUB_ACTIONS
    ? 'https://public-en-cypress.klaytn.net'
    : 'https://public-en.kairos.node.kaia.io'
  const redisClient: RedisClientType = createClient({ url: '' })

  const VALID_REPORTER = {
    id: '2',
    address: '0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266',
    privateKey: '0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80',
    oracleAddress: '0x0E4E90de7701B72df6F21343F51C833F7d2d3CFb',
    chain: '',
    service: ''
  }
  const INVALID_REPORTER = {
    id: '1',
    address: '0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266',
    privateKey: 'INVALID KEY',
    oracleAddress: '0x0E4E90de7701B72df6F21343F51C833F7d2d3CFb',
    chain: '',
    service: ''
  }

  test('Via refresh()', async function () {
    jest.spyOn(redisClient, 'set').mockImplementation(async () => {
      return null
    })
    jest
      .spyOn(State.prototype, 'all')
      .mockImplementation(async () => [VALID_REPORTER, INVALID_REPORTER])

    for (const delegatedFee of [true, false]) {
      const state = new State({
        redisClient,
        providerUrl: PROVIDER_URL,
        stateName: '',
        service: '',
        chain: '',
        delegatedFee,
        logger: buildMockLogger()
      })

      const reporters = await state.refresh()
      expect(reporters).toHaveLength(1)
    }
  })

  test('Valid reporter via add()', async function () {
    jest.spyOn(redisClient, 'set').mockImplementation(async () => {
      return null
    })
    jest.spyOn(State.prototype, 'active').mockImplementation(async () => [])
    jest.spyOn(State.prototype, 'get').mockImplementation(async () => VALID_REPORTER)

    for (const delegatedFee of [true, false]) {
      const state = new State({
        redisClient,
        providerUrl: PROVIDER_URL,
        stateName: '',
        service: '',
        chain: '',
        delegatedFee,
        logger: buildMockLogger()
      })

      const reporter = await state.add(VALID_REPORTER.id)
      expect(reporter).toEqual(VALID_REPORTER)
    }
  })

  test('Invalid reporter via add()', async function () {
    jest.spyOn(redisClient, 'set').mockImplementation(async () => {
      return null
    })
    jest.spyOn(State.prototype, 'active').mockImplementation(async () => [])
    jest.spyOn(State.prototype, 'get').mockImplementation(async () => INVALID_REPORTER)

    for (const delegatedFee of [true, false]) {
      const state = new State({
        redisClient,
        providerUrl: PROVIDER_URL,
        stateName: '',
        service: '',
        chain: '',
        delegatedFee,
        logger: buildMockLogger()
      })

      await expect(state.add(INVALID_REPORTER.id)).rejects.toThrow()
    }
  })
})
