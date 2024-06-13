import { NonceManager } from '@ethersproject/experimental'
import { Mutex } from 'async-mutex'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { getReporter, getReporters } from '../api'
import { OraklError, OraklErrorCode } from '../errors'
import { NONCE_MANAGER_POLLING_INTERVAL, NONCE_MANAGER_SLACK_FREQUENCY_RETRIES } from '../settings'
import { IReporterConfig } from '../types'
import { isAddressValid } from '../utils'
import { Wallet } from './types'
import {
  buildCaverWallet,
  buildWallet,
  CaverWallet,
  isPrivateKeyAddressPairValid,
  sleep
} from './utils'

const FILE_NAME = import.meta.url

export class State {
  redisClient: RedisClientType
  providerUrl: string
  stateName: string
  service: string
  chain: string
  delegatedFee: boolean
  logger: Logger
  wallets: { [key: string]: Wallet }
  nonces: { [key: string]: number }
  mutex: Mutex

  constructor({
    redisClient,
    providerUrl,
    stateName,
    service,
    chain,
    delegatedFee,
    logger
  }: {
    redisClient: RedisClientType
    providerUrl: string
    stateName: string
    service: string
    chain: string
    delegatedFee: boolean
    logger: Logger
  }) {
    this.redisClient = redisClient
    this.providerUrl = providerUrl
    this.stateName = stateName
    this.service = service
    this.chain = chain
    this.delegatedFee = delegatedFee
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
   * Get reporter by id.
   */
  async get(id: string) {
    this.logger.debug(`get(${id})`)
    return await getReporter({ id, logger: this.logger })
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
   * Add reporter given `id`. Reporter can be added only if it
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
      this.logger.debug({ name: 'add', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.ReporterNotAdded, msg)
    }

    const toAddReporter = await this.get(id)
    if (!toAddReporter) {
      const msg = `Reporter with ID=${id} cannot be found for service=${this.service} on chain=${this.chain}`
      this.logger.debug({ name: 'add', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.ReporterNotAdded, msg)
    }

    if (!isPrivateKeyAddressPairValid(toAddReporter.privateKey, toAddReporter.address)) {
      const msg = `Reporter with ID=${id} has invalid private key.`
      this.logger.debug({ name: 'add', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.ReporterNotAdded, msg)
    }

    if (!isAddressValid(toAddReporter.address)) {
      const msg = `Reporter with ID=${id} has invalid reporter address: ${toAddReporter.address}.`
      this.logger.debug({ name: 'add', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.ReporterNotAdded, msg)
    }

    if (!isAddressValid(toAddReporter.oracleAddress)) {
      const msg = `Reporter with ID=${id} has invalid oracle address: ${toAddReporter.oracleAddress}.`
      this.logger.debug({ name: 'add', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.ReporterNotAdded, msg)
    }

    // Update active reporters
    const updatedActiveReporters = [...activeReporters, toAddReporter]
    await this.redisClient.set(this.stateName, JSON.stringify(updatedActiveReporters))

    // Update wallets
    let wallet: Wallet
    let nonce: number
    if (this.delegatedFee) {
      wallet = buildCaverWallet({
        privateKey: toAddReporter.privateKey,
        providerUrl: this.providerUrl
      })
      nonce = Number(await wallet.caver.rpc.klay.getTransactionCount(wallet.address))
    } else {
      wallet = buildWallet({
        privateKey: toAddReporter.privateKey,
        providerUrl: this.providerUrl
      })
      nonce = await wallet.getTransactionCount()
    }
    this.wallets = { ...this.wallets, [toAddReporter.oracleAddress]: wallet }

    // Update nonce
    this.nonces = {
      ...this.nonces,
      [toAddReporter.oracleAddress]: nonce
    }

    return toAddReporter
  }

  /**
   * Remove reporter given reporter `id`. Reporter can be removed only if
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
    if (index === -1) {
      const msg = `Reporter with ID=${id} was not found.`
      this.logger.debug({ name: 'remove', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.ReporterNotFoundInState, msg)
    }

    const removedReporter = activeReporters.splice(index, 1)[0]

    const numUpdatedActiveReporters = activeReporters.length
    if (numActiveReporters == numUpdatedActiveReporters) {
      const msg = `Reporter with ID=${id} was not removed. Reporter was not found.`
      this.logger.debug({ name: 'remove', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.ReporterNotRemoved, msg)
    }

    const oracleAddress = removedReporter.oracleAddress
    if (!this.wallets[oracleAddress]) {
      const msg = `Reporter with ID=${id} was not removed. Wallet associated with ${oracleAddress} oracle was not found.`
      this.logger.debug({ name: 'remove', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.ReporterNotRemoved, msg)
    }

    // Update active reporters
    await this.redisClient.set(this.stateName, JSON.stringify(activeReporters))

    // Update wallets
    delete this.wallets[oracleAddress]

    return removedReporter
  }

  /**
   * Get all reporters for `service` and `chain` of state, and
   * activate them. Previously active reporters are deactivated.
   */
  async refresh() {
    const reporters: IReporterConfig[] = []
    this.logger.debug('refresh')

    // Fetch
    const allReporters = await this.all()

    allReporters.forEach((reporter) => {
      if (!isPrivateKeyAddressPairValid(reporter.privateKey, reporter.address)) {
        this.logger.warn(
          { name: 'refresh', file: FILE_NAME },
          `Reporter with ID=${reporter.id} has invalid private key.`
        )
      } else if (!isAddressValid(reporter.address)) {
        this.logger.warn(
          { name: 'refresh', file: FILE_NAME },
          `Reporter with ID=${reporter.id} has invalid reporter address. ${reporter.address}`
        )
      } else if (!isAddressValid(reporter.oracleAddress)) {
        this.logger.warn(
          { name: 'refresh', file: FILE_NAME },
          `Reporter with ID=${reporter.id} has invalid oracle address. ${reporter.oracleAddress}`
        )
      } else {
        reporters.push(reporter)
      }
    })

    const wallets: { [key: string]: Wallet }[] = []
    for (const R of reporters) {
      let wallet: Wallet
      let nonce: number
      if (this.delegatedFee) {
        wallet = buildCaverWallet({
          privateKey: R.privateKey,
          providerUrl: this.providerUrl
        })
        nonce = Number(await wallet.caver.rpc.klay.getTransactionCount(wallet.address))
      } else {
        wallet = buildWallet({
          privateKey: R.privateKey,
          providerUrl: this.providerUrl
        })
        nonce = await wallet.getTransactionCount()
      }

      wallets.push({ [R.oracleAddress]: wallet })

      // Update nonce
      this.nonces = {
        ...this.nonces,
        [R.oracleAddress]: nonce
      }

      // Create a mutex object
      this.mutex = new Mutex()
    }

    // Update
    await this.redisClient.set(this.stateName, JSON.stringify(reporters))
    this.wallets = Object.assign({}, ...wallets)

    return reporters
  }

  /**
   * Get nonce for oracleAddress. If nonce is not found, raise an error.
   * This function implements a mutex to ensure it cannot be called concurrently.
   *
   * @param {string} oracleAddress
   * @return {number} nonce
   * @exception {OraklErrorCode.WalletNotActive} raise when wallet is not active
   * @exception {OraklErrorCode.FailedToGetWalletTransactionCount} raise when failed to get wallet transaction count
   */
  async getAndIncrementNonce(oracleAddress: string): Promise<number> {
    return await this.mutex.runExclusive(async () => {
      const wallet = this.wallets[oracleAddress]
      if (!wallet) {
        const msg = `Wallet for oracle ${oracleAddress} is not active`
        this.logger.error({ name: 'getAndIncrementNonce', file: FILE_NAME }, msg)
        throw new OraklError(OraklErrorCode.WalletNotActive, msg)
      }

      // Assumption: the only source of error can be json-rpc call
      // Solution: keep polling/retrying until json-rpc becomes responsive
      // If successful, the "return" statement will break the infinite loop
      let retryCount = 0
      while (true) {
        try {
          let remoteNonce: number
          if (this.delegatedFee) {
            const caverWallet = wallet as CaverWallet
            remoteNonce = Number(
              await caverWallet.caver.rpc.klay.getTransactionCount(caverWallet.address)
            )
          } else {
            remoteNonce = await (wallet as NonceManager).getTransactionCount()
          }

          const localNonce = this.nonces[oracleAddress]

          let nonce: number
          if (!localNonce || remoteNonce > localNonce) {
            this.logger.warn(
              { name: 'getAndIncrementNonce', file: FILE_NAME },
              `Nonce value discrepancy. Remote: ${remoteNonce}, Local: ${localNonce}. Updating local nonce to remote nonce.`
            )
            nonce = remoteNonce
          } else {
            nonce = localNonce
          }

          this.nonces[oracleAddress] = nonce + 1

          return nonce
        } catch (error) {
          retryCount += 1

          this.logger.error(
            { name: 'getAndIncrementNonce', file: FILE_NAME },
            `Error while fetching nonce for oracle ${oracleAddress}. Retrying...`
          )

          // Slack the error message every half an hour
          if (NONCE_MANAGER_SLACK_FREQUENCY_RETRIES % retryCount === 0) {
            // Send slack message
          }

          // Sleep before retrying
          await sleep(NONCE_MANAGER_POLLING_INTERVAL)
        }
      }
    })
  }
}
