import { Queue } from 'bullmq'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { IAggregatorHeartbeatWorker } from '../types'
import { getAggregator, getAggregators } from './api'
import { OraklError, OraklErrorCode } from '../errors'
import { buildHeartbeatJobId } from '../utils'
import { HEARTBEAT_JOB_NAME, DEPLOYMENT_NAME, HEARTBEAT_QUEUE_SETTINGS } from '../settings'
import { getOperatorAddress, getSynchronizedDelay } from './data-feed.utils'
import { IAggregatorConfig } from './types'

const FILE_NAME = import.meta.url

export class State {
  redisClient: RedisClientType
  stateName: string
  heartbeatQueue: Queue
  chain: string
  logger: Logger
  wallets

  constructor({
    redisClient,
    stateName,
    heartbeatQueue,
    chain,
    logger
  }: {
    redisClient: RedisClientType
    stateName: string
    heartbeatQueue: Queue
    chain: string
    logger: Logger
  }) {
    this.redisClient = redisClient
    this.stateName = stateName
    this.chain = chain
    this.logger = logger.child({ name: 'State', file: FILE_NAME })
    this.logger.debug('Aggregator state initialized')
  }

  /**
   * Clear aggregator state.
   */
  async clear() {
    this.logger.debug('clear')
    await this.redisClient.set(this.stateName, JSON.stringify([]))
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
   *
   */
  async launchHeartbeatJob(aggregatorHash: string) {
    const aggregator = await getAggregator({
      aggregatorHash,
      chain: this.chain,
      logger: this.logger
    })

    const oracleAddress = aggregator.address
    const heartbeat = aggregator.heartbeat

    const operatorAddress = await getOperatorAddress({ oracleAddress, logger: this.logger })
    const jobData: IAggregatorHeartbeatWorker = {
      oracleAddress
    }
    await this.heartbeatQueue.add(HEARTBEAT_JOB_NAME, jobData, {
      jobId: buildHeartbeatJobId({ oracleAddress, deploymentName: DEPLOYMENT_NAME }),
      delay: await getSynchronizedDelay({
        oracleAddress,
        operatorAddress,
        heartbeat,
        logger: this.logger
      }),
      ...HEARTBEAT_QUEUE_SETTINGS
    })
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
      this.logger?.debug({ name: 'add', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.AggregatorNotAdded, msg)
    }

    const toAddAggregator = await getAggregator({
      aggregatorHash,
      chain: this.chain,
      logger: this.logger
    })
    if (!toAddAggregator || !toAddAggregator.active) {
      const msg = `Aggregator with aggregatorHash=${aggregatorHash} cannot be found / is not active on chain=${this.chain}`
      this.logger?.debug({ name: 'add', file: FILE_NAME }, msg)
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
    await this.launchHeartbeatJob(aggregatorHash)

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
      this.logger?.debug({ name: 'remove', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.AggregatorNotRemoved, msg)
    }

    // Update active aggregators
    await this.redisClient.set(this.stateName, JSON.stringify(activeAggregators))
  }
}
