import { jest } from '@jest/globals'
import { createClient, RedisClientType } from 'redis'
import { buildMockLogger } from '../src/logger'
import { State } from '../src/reporter/state'

describe('nonce-manager', () => {
  const PROVIDER_URL = process.env.GITHUB_ACTIONS
    ? 'https://public-en-cypress.klaytn.net'
    : 'https://public-en-kairos.node.kaia.io'
  const redisClient: RedisClientType = createClient({ url: '' })
  const ORACLE_ADDRESS = '0x0E4E90de7701B72df6F21343F51C833F7d2d3CFb'
  const REPORTER = {
    id: '1',
    address: '0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266',
    privateKey: '0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80',
    oracleAddress: ORACLE_ADDRESS,
    chain: '',
    service: '',
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
      logger: buildMockLogger(),
    })

    delegatedState = new State({
      redisClient,
      providerUrl: PROVIDER_URL,
      stateName: '',
      service: '',
      chain: '',
      delegatedFee: true,
      logger: buildMockLogger(),
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

  test('concurrent nonce calls', async () => {
    // send multiple concurrent calls to getAndIncrementNonce()
    // check that all nonces are unique and increment by 1
    const CONCURRENT_CALLS = 50

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
})
