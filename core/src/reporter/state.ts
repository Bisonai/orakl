import { Logger } from 'pino'
import { ethers } from 'ethers'
import type { RedisClientType } from 'redis'
import { getReporters, getReporter } from '../api'
import { IReporterConfig } from '../types'
import { OraklError, OraklErrorCode } from '../errors'
import { buildWallet } from './utils'

const FILE_NAME = import.meta.url

export class State {
  redisClient: RedisClientType
  providerUrl: string
  stateName: string
  service: string
  chain: string
  logger: Logger
  wallets

  constructor({
    redisClient,
    providerUrl,
    stateName,
    service,
    chain,
    logger
  }: {
    redisClient: RedisClientType
    providerUrl: string
    stateName: string
    service: string
    chain: string
    logger: Logger
  }) {
    this.redisClient = redisClient
    this.providerUrl = providerUrl
    this.stateName = stateName
    this.service = service
    this.chain = chain
    this.logger = logger.child({ name: 'State', file: FILE_NAME })
    this.logger.debug('Reporter state initialized')
  }

  /**
   * Clear reporter state.
   */
  async clear() {
    this.logger.debug('clear')
    await this.redisClient.set(this.stateName, JSON.stringify([]))
    this.logger.debug('Reporter state cleared')
  }

  /**
   * List all reporters given `service` and `chain`. The reporters can
   * be either active or inactive.
   */
  async all() {
    this.logger.debug('all')
    return await getReporters({ service: this.service, chain: this.chain, logger: this.logger })
  }

  /**
   * List all active reporters.
   */
  async active() {
    this.logger.debug('active')
    const state = await this.redisClient.get(this.stateName)
    return state ? JSON.parse(state) : []
  }

  /**
   * Add reporter based given `id`. Reporter can be added only if it
   * corresponds to the `service` and `chain` state.
   *
   * @param {string} reporter ID
   * @return {IReporterConfig}
   * @exception {OraklErrorCode.ReporterNotAdded} raise when no reporter was added
   */
  async add(id: string): Promise<IReporterConfig> {
    this.logger.debug('add')

    // Check if reporter is not active yet
    const activeReporters = await this.active()
    const isAlreadyActive = activeReporters.filter((L) => L.id === id) || []

    if (isAlreadyActive.length > 0) {
      const msg = `Reporter with ID=${id} was not added. It is already active.`
      this.logger?.debug({ name: 'add', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.ReporterNotAdded, msg)
    }

    const toAddReporter = await getReporter({ id, logger: this.logger })
    if (!toAddReporter) {
      const msg = `Reporter with ID=${id} cannot be found for service=${this.service} on chain=${this.chain}`
      this.logger?.debug({ name: 'add', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.ReporterNotAdded, msg)
    }

    // Update active reporters
    const updatedActiveReporters = [...activeReporters, toAddReporter]
    await this.redisClient.set(this.stateName, JSON.stringify(updatedActiveReporters))

    // Update wallets
    const wallet = await buildWallet({
      privateKey: toAddReporter.privateKey,
      providerUrl: this.providerUrl
    })
    this.wallets = { ...this.wallets, [toAddReporter.oracleAddress]: wallet }

    return toAddReporter
  }

  /**
   * Remove reporter given reporter `id`. Reporter can removed only if
   * it was in an active state.
   *
   * @param {string} reporter ID
   * @exception {OraklErrorCode.ReporterNotRemoved} raise when no reporter was removed
   */
  async remove(id: string) {
    this.logger.debug('remove')

    const activeReporters = await this.active()
    const numActiveReporters = activeReporters.length

    const index = activeReporters.findIndex((L) => L.id == id)
    const removedReporter = activeReporters.splice(index, 1)[0]

    const numUpdatedActiveReporters = activeReporters.length
    if (numActiveReporters == numUpdatedActiveReporters) {
      const msg = `Reporter with ID=${id} was not removed. Reporter was not found.`
      this.logger?.debug({ name: 'remove', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.ReporterNotRemoved, msg)
    }

    const oracleAddress = removedReporter.oracleAddress
    if (!this.wallets[oracleAddress]) {
      const msg = `Reporter with ID=${id} was not removed. Wallet associated with ${oracleAddress} oracle was not found.`
      this.logger?.debug({ name: 'remove', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.ReporterNotRemoved, msg)
    }

    // Update active reporters
    await this.redisClient.set(this.stateName, JSON.stringify(activeReporters))

    // Update wallets
    delete this.wallets[oracleAddress]
  }

  /**
   * Get all reporters for `service` and `chain` of state, and
   * activate them. Previously active reporters are deactivated.
   */
  async refresh() {
    this.logger.debug('refresh')

    // Fetch
    const reporters = await this.all()
    const wallets = reporters.map((R) => {
      const W = buildWallet({
        privateKey: R.privateKey,
        providerUrl: this.providerUrl
      })
      return { [R.oracleAddress]: W }
    })

    // Update
    await this.redisClient.set(this.stateName, JSON.stringify(reporters))
    this.wallets = Object.assign({}, ...wallets)

    return reporters
  }
}
