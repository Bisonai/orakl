import * as path from 'node:path'
import { Queue } from 'bullmq'
import { Contract, ethers } from 'ethers'
import { Logger } from 'pino'
import { PROVIDER_URL, BULLMQ_CONNECTION, LISTENER_ROOT_DIR, LISTENER_DELAY } from '../settings'
import { IListenerBlock, IListenerConfig } from '../types'
import { mkdir, readTextFile, writeTextFile } from '../utils'

export class Event {
  fn: (log) => void
  emitContract: Contract
  listenerBlock: IListenerBlock
  provider: ethers.providers.JsonRpcProvider
  eventName: string
  running: boolean
  logger: Logger

  constructor(
    queueName: string,
    wrapFn: (iface: ethers.utils.Interface, queue: Queue, logger: Logger) => (log) => void,
    abi,
    listener: IListenerConfig,
    logger
  ) {
    this.logger = logger

    this.logger.debug(`listenToEvents:topicId ${listener.eventName}`)
    this.logger.debug(`PROVIDER_URL ${PROVIDER_URL}`)
    this.logger.debug(`LISTENER_ROOT_DIR ${LISTENER_ROOT_DIR}`)

    mkdir(LISTENER_ROOT_DIR)
    const provider = new ethers.providers.JsonRpcProvider(PROVIDER_URL)
    const iface = new ethers.utils.Interface(abi)
    const queue = new Queue(queueName, BULLMQ_CONNECTION)

    this.running = false
    this.provider = provider
    this.fn = wrapFn(iface, queue, logger)
    this.eventName = listener.eventName
    this.emitContract = new ethers.Contract(listener.address, abi, provider)
    this.listenerBlock = {
      startBlock: 0,
      filePath: path.join(LISTENER_ROOT_DIR, `${listener.address}.txt`)
    }
  }

  listen() {
    setInterval(async () => {
      if (!this.running) {
        this.running = true
        await this.filter()
        this.running = false
      } else {
        this.logger.debug('running')
      }
    }, LISTENER_DELAY)
  }

  async filter() {
    try {
      if (this.listenerBlock.startBlock == 0) {
        try {
          this.listenerBlock.startBlock = parseInt(await readTextFile(this.listenerBlock.filePath))
        } catch {
          this.listenerBlock.startBlock = await this.getLatestBlock()
        }
      }

      const latestBlock = await this.getLatestBlock()
      if (latestBlock >= this.listenerBlock.startBlock) {
        const events = await this.emitContract.queryFilter(
          this.eventName,
          this.listenerBlock.startBlock,
          latestBlock
        )

        this.logger.debug(`${this.listenerBlock.startBlock}-${latestBlock}`)
        this.listenerBlock.startBlock = latestBlock + 1
        await writeTextFile(this.listenerBlock.filePath, this.listenerBlock.startBlock.toString())

        if (events?.length > 0) {
          events.forEach(this.fn)
        }
      }
    } catch (e) {
      this.logger.error(e)
    }
  }

  async getLatestBlock() {
    return await this.provider.getBlockNumber()
  }
}
