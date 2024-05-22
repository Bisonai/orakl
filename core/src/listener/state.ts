import { Queue } from 'bullmq'
import ethers from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { OraklError, OraklErrorCode } from '../errors'
import {
  getObservedBlockRedisKey,
  LISTENER_DELAY,
  LISTENER_JOB_SETTINGS,
  PROVIDER_URL
} from '../settings'
import { IListenerConfig, IListenerRawConfig } from '../types'
import { getListeners, getObservedBlock, upsertObservedBlock } from './api'
import { IContracts, ILatestListenerJob } from './types'
import { postprocessListeners } from './utils'

const FILE_NAME = import.meta.url

/**
 * Listener's ephemeral state holds information about all tracked
 * events, and allows to activate/deactive any event while the
 * listener service is running.
 *
 * Listener's ephemeral state includes read & write state, but also
 * auxiliary operational functions.
 *
 * Read functions:
 * - all()
 * - active()

 * Write functions:
 * - clear()
 * - add(id: string)
 * - remove(id: string)

 * Operational functions:
 * - queryEvent(contractAddress: string, fromBlockNumber: number, toBlockNumber: number)
 * - latestBlockNumber()
 */
export class State {
  redisClient: RedisClientType
  latestListenerQueue: Queue
  historyListenerQueue: Queue
  stateName: string
  service: string
  chain: string
  eventName: string
  logger: Logger
  provider: ethers.providers.JsonRpcProvider
  contracts: IContracts
  abi: ethers.ContractInterface

  constructor({
    redisClient,
    latestListenerQueue,
    historyListenerQueue,
    stateName,
    service,
    chain,
    eventName,
    abi,
    logger
  }: {
    redisClient: RedisClientType
    latestListenerQueue: Queue
    historyListenerQueue: Queue
    stateName: string
    service: string
    chain: string
    eventName: string
    abi: ethers.ContractInterface
    logger: Logger
  }) {
    this.redisClient = redisClient
    this.latestListenerQueue = latestListenerQueue
    this.historyListenerQueue = historyListenerQueue
    this.stateName = stateName
    this.service = service
    this.abi = abi
    this.chain = chain
    this.eventName = eventName
    this.logger = logger
    this.provider = new ethers.providers.JsonRpcProvider(PROVIDER_URL)
    this.contracts = {}
  }

  /**
   * Clear listener state.
   */
  async clear(): Promise<void> {
    this.logger.debug('State.clear')
    const activeListeners = await this.active()
    for (const listener of activeListeners) {
      await this.remove(listener.id)
    }

    await this.redisClient.set(this.stateName, JSON.stringify([]))

    const jobs = await this.latestListenerQueue.getRepeatableJobs()
    await Promise.all(
      jobs.map((J) => {
        this.latestListenerQueue.removeRepeatableByKey(J.key)
      })
    )
  }

  /**
   * List all listeners given `service` and `chain`. The listeners can
   * be either active or inactive.
   *
   * @return {Promise<IListenerRawConfig[]>} list of all listeners from permanent state
   * @exception {GetListenerRequestFailed}
   */
  async all(): Promise<IListenerRawConfig[]> {
    this.logger.debug('State.all')
    return await getListeners({ service: this.service, chain: this.chain, logger: this.logger })
  }

  /**
   * List all active listeners.
   *
   * @return {Promise<IListenerConfig[]>} list of active listeners from ephemeral state
   */
  async active(): Promise<IListenerConfig[]> {
    this.logger.debug('State.active')
    const state = await this.redisClient.get(this.stateName)
    return state ? JSON.parse(state) : []
  }

  /**
   * Add listener identified by `id` parameter. `id` is a global
   * listener identifier stored at permanent listener state. Listener
   * can be added only if it is associated with the `service` and `chain`.
   *
   * @param {string} listener ID
   * @return {IListenerConfig}
   * @exception {OraklErrorCode.ListenerNotAdded} raise when no listener was added
   */
  async add(id: string): Promise<IListenerConfig> {
    this.logger.debug('State.add')

    // Check if listener is not active yet
    const activeListeners = await this.active()
    const isAlreadyActive = activeListeners.filter((L) => L.id === id) || []

    if (isAlreadyActive.length > 0) {
      const msg = `Listener with ID=${id} was not added. It is already active.`
      this.logger.warn({ name: 'add', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.ListenerNotAdded, msg)
    }

    // Find listener by ID
    const listenersRawConfig = await this.all()
    const allListeners = postprocessListeners({
      listenersRawConfig,
      service: this.service,
      chain: this.chain,
      logger: this.logger
    })

    const allServiceListeners = allListeners[this.service] || []
    const filteredListeners = allServiceListeners.filter((L) => L.id === id)

    if (filteredListeners.length != 1) {
      const msg = `Listener with ID=${id} cannot be found for service=${this.service} on chain=${this.chain}`
      this.logger.error({ name: 'add', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.ListenerNotAdded, msg)
    }

    // Update active listeners
    const toAddListener = filteredListeners[0]
    const updatedActiveListeners = [...activeListeners, toAddListener]
    await this.redisClient.set(this.stateName, JSON.stringify(updatedActiveListeners))

    const contractAddress = toAddListener.address
    const observedBlockRedisKey = getObservedBlockRedisKey(contractAddress)
    const latestBlock = await this.latestBlockNumber()
    const { blockKey: observedBlockKey } = await getObservedBlock({
      blockKey: observedBlockRedisKey,
      logger: this.logger
    })

    /**
     when listener starts, there are two options:
      * latest observedBlock (key) exists -> do nothing 
        (start latest job, it'll handle multiple blocks 
        between observedBlock and latestBlock)
      * it does not exist -> upsert latestBlock
     */
    if (observedBlockKey === '') {
      await upsertObservedBlock({
        blockKey: observedBlockRedisKey,
        blockNumber: Math.max(latestBlock - 1, 0),
        logger: this.logger
      })
      await this.redisClient.set(observedBlockRedisKey, Math.max(latestBlock - 1, 0))
    }

    // Insert listener jobs
    const outData: ILatestListenerJob = {
      contractAddress
    }
    await this.latestListenerQueue.add('latest-repeatable', outData, {
      ...LISTENER_JOB_SETTINGS,
      jobId: contractAddress,
      repeat: {
        every: LISTENER_DELAY
      }
    })

    const contract = new ethers.Contract(contractAddress, this.abi, this.provider)
    this.contracts = { ...this.contracts, [contractAddress]: contract }

    return toAddListener
  }

  /**
   * Remove listener identified by `id` parameter. `id` is a global
   * listener identifier stored at permanent listener state. Listener
   * can be removed only if it is in an active state.
   *
   * @param {string} listener ID
   * @exception {OraklErrorCode.ListenerNotRemoved} raise when no listener was removed
   */
  async remove(id: string) {
    this.logger.debug('State.remove')

    const activeListeners = await this.active()
    const numActiveListeners = activeListeners.length

    const index = activeListeners.findIndex((L) => L.id === id)
    if (index === -1) {
      const msg = `Listener with ID=${id} was not found.`
      this.logger.warn({ name: 'remove', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.ListenerNotFoundInState, msg)
    }

    const removedListener = activeListeners.splice(index, 1)[0]

    const jobs = (await this.latestListenerQueue.getRepeatableJobs()).filter(
      (job) => job.id == removedListener.address
    )

    if (jobs.length != 1) {
      // throw new OraklError(
      //   OraklErrorCode.UnexpectedNumberOfJobsInQueue,
      //   `Number of jobs ${jobs.length}`
      // )
    } else {
      const delayedJob = jobs[0]
      await this.latestListenerQueue.removeRepeatableByKey(delayedJob.key)

      this.logger.debug({ job: 'deleted' }, `Listener delayed job with KEY=${delayedJob.key}`)
    }

    const numUpdatedActiveListeners = activeListeners.length
    if (numActiveListeners === numUpdatedActiveListeners) {
      const msg = `Listener with ID=${id} was not removed.`
      this.logger.error({ name: 'remove', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.ListenerNotRemoved, msg)
    }

    // Update active listeners
    await this.redisClient.set(this.stateName, JSON.stringify(activeListeners))

    // Update active contracts
    delete this.contracts[removedListener.address]

    return removedListener
  }

  /**
   * Query event on smart contract (specified by `contractAddress`) from
   * blocks in between `fromBlockNumber` and `toBlockNumber`.
   */
  async queryEvent(contractAddress: string, fromBlockNumber: number, toBlockNumber: number) {
    return await this.contracts[contractAddress].queryFilter(
      this.eventName,
      fromBlockNumber,
      toBlockNumber
    )
  }

  /**
   * Fetch the latest block number.
   */
  async latestBlockNumber() {
    return await this.provider.getBlockNumber()
  }

  /**
   * Set the observed block number (`observedBlockNumber`) if it has
   * not been defined yet.
   *
   * @param {string} redis key that points to informatation about the latest observed block
   * @param {number} observed block number to use if none defined
   */
  async setObservedBlockNumberIfNotDefined(
    observedBlockRedisKey: string,
    observedBlockNumber: number
  ) {
    if ((await this.redisClient.get(observedBlockRedisKey)) === null) {
      await this.redisClient.set(observedBlockRedisKey, observedBlockNumber)
    }
  }
}
