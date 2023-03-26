import { Queue } from 'bullmq'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { IAggregatorSubmitHeartbeatWorker } from '../types'
import { getAggregator, getAggregators } from './api'
import { OraklError, OraklErrorCode } from '../errors'
import { SUBMIT_HEARTBEAT_QUEUE_SETTINGS } from '../settings'
import { IAggregatorConfig } from './types'

const FILE_NAME = import.meta.url

export class State {
  redisClient: RedisClientType
  stateName: string
  heartbeatQueue: Queue
  submitHeartbeatQueue: Queue
  chain: string
  logger: Logger
  wallets

  constructor({
    redisClient,
    stateName,
    heartbeatQueue,
    submitHeartbeatQueue,
    chain,
    logger
  }: {
    redisClient: RedisClientType
    stateName: string
    heartbeatQueue: Queue
    submitHeartbeatQueue: Queue
    chain: string
    logger: Logger
  }) {
    this.redisClient = redisClient
    this.stateName = stateName
    this.heartbeatQueue = heartbeatQueue
    this.submitHeartbeatQueue = submitHeartbeatQueue
    this.chain = chain
    this.logger = logger.child({ name: 'State', file: FILE_NAME })
    this.logger.debug('Aggregator state initialized')
  }

  /**
   * Clear aggregator state.
   */
  async clear() {
    this.logger.debug('clear')

    // Clear aggregator ephemeral state
    await this.redisClient.set(this.stateName, JSON.stringify([]))

    // Remove previously launched heartbeat jobs
    const delayedJobs = await this.heartbeatQueue.getJobs(['delayed'])
    for (const job of delayedJobs) {
      await job.remove()
    }

    this.logger.debug('Aggregator state cleared')
  }

  /**
   * List all aggregators given `chain`. The aggregator can
   * be either active or inactive.
   */
  async all() {
    this.logger.debug('all')
    return await getAggregators({ chain: this.chain, logger: this.logger })
  }

  /**
   * List all aggregators in an active state.
   */
  async active() {
    this.logger.debug('active')
    const state = await this.redisClient.get(this.stateName)
    return state ? JSON.parse(state) : []
  }

  /**
   * Check whether given `oracleAddress` is active in local state or
   * not.
   *
   * @param {string} oracleAddress
   */
  async isActive({ oracleAddress }: { oracleAddress: string }) {
    this.logger.debug('isActive')
    const activeAggregators = await this.active()
    const isAlreadyActive = activeAggregators.filter((L) => L.address === oracleAddress) || []

    if (isAlreadyActive.length > 0) {
      return true
    } else {
      return false
    }
  }

  /**
   * Add aggregator given `aggregatorHash`. Aggregator can be added only if it
   * corresponds to the `chain` state.
   *
   * @param {string} aggregator hash
   * @return {IAggregatorConfig}
   * @exception {OraklErrorCode.AggregatorNotAdded} raise when no aggregator was added
   */
  async add(aggregatorHash: string): Promise<IAggregatorConfig> {
    this.logger.debug('add')

    // Check if reporter is not active in service yet
    const activeAggregators = await this.active()

    // TODO store in dictionary instead
    const isAlreadyActive =
      activeAggregators.filter((L) => L.aggregatorHash === aggregatorHash) || []

    if (isAlreadyActive.length > 0) {
      const msg = `Aggregator with aggregatorHash=${aggregatorHash} was not added. It is already active.`
      this.logger.debug({ name: 'add', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.AggregatorNotAdded, msg)
    }

    const toAddAggregator = await getAggregator({
      aggregatorHash,
      chain: this.chain,
      logger: this.logger
    })
    if (!toAddAggregator || !toAddAggregator.active) {
      const msg = `Aggregator with aggregatorHash=${aggregatorHash} cannot be found / is not active on chain=${this.chain}`
      this.logger.debug({ name: 'add', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.AggregatorNotAdded, msg)
    }

    // Update active aggregators
    const aggregatorConfig: IAggregatorConfig = {
      id: toAddAggregator.id.toString(),
      aggregatorHash: toAddAggregator.aggregatorHash,
      name: toAddAggregator.name,
      address: toAddAggregator.address,
      heartbeat: toAddAggregator.heartbeat,
      threshold: toAddAggregator.threshold,
      absoluteThreshold: toAddAggregator.absoluteThreshold,
      chain: this.chain
    }
    await this.redisClient.set(
      this.stateName,
      JSON.stringify([...activeAggregators, aggregatorConfig])
    )

    const outDataSubmitHeartbeat: IAggregatorSubmitHeartbeatWorker = {
      oracleAddress: toAddAggregator.address,
      delay: toAddAggregator.heartbeat
    }
    this.logger.debug(outDataSubmitHeartbeat, 'outDataSubmitHeartbeat')
    await this.submitHeartbeatQueue.add('state-submission', outDataSubmitHeartbeat, {
      ...SUBMIT_HEARTBEAT_QUEUE_SETTINGS
    })

    return aggregatorConfig
  }

  /**
   * Remove aggregator given `aggregatorHash`. Aggregator can be removed only if
   * it was in an active state.
   *
   * @param {string} aggregator hash
   * @exception {OraklErrorCode.AggregatorNotRemoved} raise when no reporter was removed
   */
  async remove(aggregatorHash: string) {
    this.logger.debug('remove')

    const activeAggregators = await this.active()
    const numActiveAggregators = activeAggregators.length

    const index = activeAggregators.findIndex((L) => L.aggregatorHash == aggregatorHash)
    const removedAggregator = activeAggregators.splice(index, 1)[0]

    const numUpdatedActiveAggregators = activeAggregators.length
    if (numActiveAggregators == numUpdatedActiveAggregators) {
      const msg = `Aggregator with aggregatorHash=${aggregatorHash} was not removed. Aggregator was not found.`
      this.logger.debug({ name: 'remove', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.AggregatorNotRemoved, msg)
    }

    // Update active aggregators
    await this.redisClient.set(this.stateName, JSON.stringify(activeAggregators))

    return removedAggregator
  }
}
