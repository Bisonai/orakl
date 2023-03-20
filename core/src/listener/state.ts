import { Logger } from 'pino'
import { getListeners } from './api'
import { postprocessListeners } from './utils'
import { IListenerConfig } from '../types'
import { OraklError, OraklErrorCode } from '../errors'

const FILE_NAME = import.meta.url

export class State {
  redisClient
  listenerStateName: string
  service: string
  chain: string
  logger: Logger

  constructor({
    redisClient,
    listenerStateName,
    service,
    chain,
    logger
  }: {
    redisClient
    listenerStateName: string
    service: string
    chain: string
    logger: Logger
  }) {
    this.redisClient = redisClient
    this.listenerStateName = listenerStateName
    this.service = service
    this.chain = chain
    this.logger = logger
  }

  /**
   * Initialize a state with multiple listener configurations at
   * once. This method is expected to be called once, after the object
   * is initialized.
   *
   * @param {IListenerConfig[]} list of listener configurations
   */
  async init(config: IListenerConfig[]) {
    await this.redisClient.set(this.listenerStateName, JSON.stringify(config))
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
    return JSON.parse(await this.redisClient.get(this.listenerStateName))
  }

  /**
   * Add listener based given `id`. Listener can be added only if it
   * corresponds to the `service` and `chain` state.
   *
   * @param {string} listener ID
   * @exception {OraklErrorCode.ListenerNotAdded} raise when no listener was added
   */
  async add(id: string) {
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
    await this.redisClient.set(this.listenerStateName, JSON.stringify(updatedActiveListeners))

    return toAddListener[0]
  }

  /**
   * Remove listener based given `id`. Listener can removed only if
   * it was in an active state.
   *
   * @param {string} listener ID
   * @exception {OraklErrorCode.ListenerNotRemoved} raise when no listener removed
   */
  async remove(id: string) {
    const activeListeners = await this.active()

    const updatedActiveListeners = activeListeners.filter((L) => L.id != id)

    const numActiveListeners = activeListeners.length
    const numUpdatedActiveListeners = updatedActiveListeners.length
    if (numActiveListeners == numUpdatedActiveListeners) {
      const msg = `Listener with ID=${id} was not removed.`
      this.logger?.debug({ name: 'remove', file: FILE_NAME }, msg)
      throw new OraklError(OraklErrorCode.ListenerNotRemoved, msg)
    }

    await this.redisClient.set(this.listenerStateName, JSON.stringify(updatedActiveListeners))
  }
}
