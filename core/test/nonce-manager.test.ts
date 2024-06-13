import { NonceManager } from '@ethersproject/experimental'
import { jest } from '@jest/globals'
import { Mutex } from 'async-mutex'
import { createClient, RedisClientType } from 'redis'
import { OraklError, OraklErrorCode } from '../src/errors'
import { buildMockLogger } from '../src/logger'
import { State } from '../src/reporter/state'
import { CaverWallet } from '../src/reporter/utils'

describe('nonce-manager', () => {
  const PROVIDER_URL = process.env.GITHUB_ACTIONS
    ? 'https://public-en-cypress.klaytn.net'
    : 'https://public-en.kairos.node.kaia.io'
  const redisClient: RedisClientType = createClient({ url: '' })
  const ORACLE_ADDRESS = '0x0E4E90de7701B72df6F21343F51C833F7d2d3CFb'
  const REPORTER = {
    id: '1',
    address: '0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266',
    privateKey: '0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80',
    oracleAddress: ORACLE_ADDRESS,
    chain: '',
    service: ''
  }

  jest.spyOn(State.prototype, 'all').mockImplementation(async () => [REPORTER])
  jest.spyOn(redisClient, 'set').mockImplementation(async () => {
    return null
  })

  let state: State
  let delegatedState: State

  beforeEach(() => {
    state = new State({
      redisClient,
      providerUrl: PROVIDER_URL,
      stateName: '',
      service: '',
      chain: '',
      delegatedFee: false,
      logger: buildMockLogger()
    })

    delegatedState = new State({
      redisClient,
      providerUrl: PROVIDER_URL,
      stateName: '',
      service: '',
      chain: '',
      delegatedFee: true,
      logger: buildMockLogger()
    })
  })

  test('wallet not active', async () => {
    try {
      state.mutex = new Mutex()
      state.wallets = {}
      await state.getAndIncrementNonce(ORACLE_ADDRESS)
    } catch (error) {
      expect(error.code).toBe(OraklErrorCode.WalletNotActive)
    }
  })

  test('cannot get transaction count', async () => {
    // override state.getTransactionCount() to throw error
    await state.refresh()
    const wallet = state.wallets[ORACLE_ADDRESS] as NonceManager
    jest.spyOn(wallet, 'getTransactionCount').mockImplementation(async () => {
      throw new OraklError(OraklErrorCode.FailedToGetWalletTransactionCount)
    })
    try {
      await state.getAndIncrementNonce(ORACLE_ADDRESS)
    } catch (error) {
      expect(error.code).toBe(OraklErrorCode.FailedToGetWalletTransactionCount)
    }

    await delegatedState.refresh()
    const caverWallet = delegatedState.wallets[ORACLE_ADDRESS] as CaverWallet
    jest.spyOn(caverWallet.caver.rpc.klay, 'getTransactionCount').mockImplementation(async () => {
      throw new OraklError(OraklErrorCode.FailedToGetWalletTransactionCount)
    })
  })

  test('increments nonce after func is called', async () => {
    for (const currState of [state, delegatedState]) {
      await currState.refresh()
      const returnedNonce = await currState.getAndIncrementNonce(ORACLE_ADDRESS)
      const currNonce = currState.nonces[ORACLE_ADDRESS]
      expect(currNonce).toBe(returnedNonce + 1)
    }
  })

  test('check delegatedFee handling', async () => {
    // check that nonce is updated correctly when delegatedFee is true & false
    await state.refresh()
    const wallet = state.wallets[ORACLE_ADDRESS] as NonceManager
    const currNonce = await wallet.getTransactionCount()
    const returnedNonce = await state.getAndIncrementNonce(ORACLE_ADDRESS)
    expect(returnedNonce).toBe(currNonce)

    await delegatedState.refresh()
    const caverWallet = delegatedState.wallets[ORACLE_ADDRESS] as CaverWallet
    const currCaverNonce = Number(
      await caverWallet.caver.rpc.klay.getTransactionCount(caverWallet.address)
    )
    const returnedCaverNonce = await delegatedState.getAndIncrementNonce(ORACLE_ADDRESS)
    expect(returnedCaverNonce).toBe(currCaverNonce)
  })

  test('concurrent nonce calls', async () => {
    // send multiple concurrent calls to getAndIncrementNonce()
    // check that all nonces are unique and increment by 1
    const CONCURRENT_CALLS = 10

    for (const currState of [state, delegatedState]) {
      await currState.refresh()
      const noncePromises: Promise<number>[] = []
      for (let i = 0; i < CONCURRENT_CALLS; i++) {
        noncePromises.push(state.getAndIncrementNonce(ORACLE_ADDRESS))
      }
      const nonces = await Promise.all(noncePromises)
      expect(new Set(nonces).size).toBe(CONCURRENT_CALLS)
      expect(Math.max(...nonces) - Math.min(...nonces)).toBe(CONCURRENT_CALLS - 1)
    }
  })

  test('localNonce is smaller than walletNonce', async () => {
    // when walletNonce is greater than localNonce,
    // localNonce should be updated to walletNonce
    for (const currState of [state, delegatedState]) {
      await currState.refresh()
      currState.nonces[ORACLE_ADDRESS] = 0
      const nonce = await currState.getAndIncrementNonce(ORACLE_ADDRESS)

      if (!currState.delegatedFee) {
        const wallet = currState.wallets[ORACLE_ADDRESS] as NonceManager
        const walletNonce = await wallet.getTransactionCount()
        expect(nonce).toBe(walletNonce)
      } else {
        const caverWallet = currState.wallets[ORACLE_ADDRESS] as CaverWallet
        const walletNonce = Number(
          await caverWallet.caver.rpc.klay.getTransactionCount(caverWallet.address)
        )
        expect(nonce).toBe(walletNonce)
      }
    }
  })

  test('localNonce is greater than walletNonce', async () => {
    // when localNonce is smaller than walletNonce,
    // nothing should happen, localNonce should be returned and incremented
    for (const currState of [state, delegatedState]) {
      await currState.refresh()

      if (!currState.delegatedFee) {
        const wallet = currState.wallets[ORACLE_ADDRESS] as NonceManager
        const walletNonce = await wallet.getTransactionCount()

        currState.nonces[ORACLE_ADDRESS] = walletNonce + 1
        const nonce = await currState.getAndIncrementNonce(ORACLE_ADDRESS)
        expect(nonce).not.toBe(walletNonce)
      } else {
        const caverWallet = currState.wallets[ORACLE_ADDRESS] as CaverWallet
        const walletNonce = Number(
          await caverWallet.caver.rpc.klay.getTransactionCount(caverWallet.address)
        )

        currState.nonces[ORACLE_ADDRESS] = walletNonce + 1
        const nonce = await currState.getAndIncrementNonce(ORACLE_ADDRESS)
        expect(nonce).not.toBe(walletNonce)
      }
    }
  })
})
