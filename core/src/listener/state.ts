import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { getListeners } from './api'
import { postprocessListeners } from './utils'
import { IListenerConfig } from '../types'
import { OraklError, OraklErrorCode } from '../errors'

const FILE_NAME = import.meta.url

export class State {
  redisClient: RedisClientType
  stateName: string
  service: string
  chain: string
  logger: Logger

  constructor({
    redisClient,
    stateName,
    service,
    chain,
    logger
  }: {
    redisClient: RedisClientType
    stateName: string
    service: string
    chain: string
    logger: Logger
  }) {
    this.redisClient = redisClient
    this.stateName = stateName
    this.service = service
    this.chain = chain
    this.logger = logger
  }

  /**
   * Clear listener state.
   */
  async clear() {
    await this.redisClient.set(this.stateName, JSON.stringify([]))
  }

  /**
   * List all listeners given `service` and `chain`. The listeners can
   * be either active or inactive.
   */
  async all() {
    return await getListeners({ service: this.service, chain: this.chain, logger: this.logger })
  }

  /**
   * List all active listeners.
   */
  async active() {
    const state = await this.redisClient.get(this.stateName)
    return state ? JSON.parse(state) : []
  }

  /**
   * Update the listener defined as IListenerConfig with
   * `intervalId`. `intervalId` is used to `clearInterval`.
   *
   * @param {string} listener ID
   * @param {number} interval ID
   */
  async update(id: string, intervalId: number) {
    const activeListeners = await this.active()

    const index = activeListeners.findIndex((L) => L.id == id)
    const updatedListener = activeListeners.splice(index, 1)[0]

    const updatedActiveListeners = [...activeListeners, { ...updatedListener, intervalId }]
    await this.redisClient.set(this.stateName, JSON.stringify(updatedActiveListeners))
  }

  /**
   * Add listener based given `id`. Listener can be added only if it
   * corresponds to the `service` and `chain` state.
   *
   * @param {string} listener ID
   * @return {IListenerConfig}
   * @exception {OraklErrorCode.ListenerNotAdded} raise when no listener was added
   */
  async add(id: string): Promise<IListenerConfig> {
    // Check if listener is not active yet
    const activeListeners = await this.active()
    const isAlreadyActive = activeListeners.filter((L) => L.id === id) || []

    if (isAlreadyActive.length > 0) {
      const msg = `Listener with ID=${id} was not added. It is already active.`
      this.logger?.debug({ name: 'add', file: FILE_NAME }, msg)
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
    const toAddListener = allServiceListeners.filter((L) => L.id == id)

    if (toAddListener.length != 1) {
      const msg = `Listener with ID=${id} cannot be found for service=${this.service} on chain=${this.chain}`
      this.logger?.debug({ name: 'add', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.ListenerNotAdded, msg)
    }

    // Update active listeners
    const updatedActiveListeners = [...activeListeners, ...toAddListener]
    await this.redisClient.set(this.stateName, JSON.stringify(updatedActiveListeners))

    return toAddListener[0]
  }

  /**
   * Remove listener given listener `id`. Listener can removed only if
   * it was in an active state.
   *
   * @param {string} listener ID
   * @exception {OraklErrorCode.ListenerNotRemoved} raise when no listener was removed
   */
  async remove(id: string) {
    const activeListeners = await this.active()
    const numActiveListeners = activeListeners.length

    const index = activeListeners.findIndex((L) => L.id == id)
    const removedListener = activeListeners.splice(index, 1)[0]

    const numUpdatedActiveListeners = activeListeners.length
    if (numActiveListeners == numUpdatedActiveListeners) {
      const msg = `Listener with ID=${id} was not removed.`
      this.logger?.debug({ name: 'remove', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.ListenerNotRemoved, msg)
    }

    // Update active listeners
    await this.redisClient.set(this.stateName, JSON.stringify(activeListeners))
    clearInterval(removedListener.intervalId)
  }
}
