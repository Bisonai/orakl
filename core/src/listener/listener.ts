import { Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import { PROVIDER_URL, BULLMQ_CONNECTION, LISTENER_DELAY } from '../settings'
import { IListenerConfig } from '../types'

export function listen({
  queueName,
  processEventFn,
  abi,
  redisClient,
  logger
}: {
  queueName: string
  processEventFn
  abi
  redisClient
  logger: Logger
}) {
  async function wrapper(listener: IListenerConfig): Promise<number> {
    const provider = new ethers.providers.JsonRpcProvider(PROVIDER_URL)
    const contract = new ethers.Contract(listener.address, abi, provider)
    const iface = new ethers.utils.Interface(abi)
    const queue = new Queue(queueName, BULLMQ_CONNECTION)
    const processEvent = processEventFn(iface, queue, logger)
    const listenerRedisKey = `listener:${listener.id}`

    let observedBlock = (await redisClient.get(listenerRedisKey)) || 0

    const intervalObj = setInterval(async () => {
      try {
        const latestBlock = await provider.getBlockNumber()
        logger.debug(`latest: ${latestBlock}, observedBlock: ${observedBlock}`)
        if (latestBlock > observedBlock) {
          const events = await contract.queryFilter(listener.eventName, observedBlock, latestBlock)

          if (events?.length > 0) {
            logger.debug({ name: 'listen:wrapper' }, `${events}`)
            events.forEach(processEvent)
          }
        }

        observedBlock = latestBlock
        await redisClient.set(listenerRedisKey, observedBlock)
      } catch (e) {
        logger.error({ name: 'listen:wrapper' }, e)
      }
    }, LISTENER_DELAY)

    const intervalId = intervalObj[Symbol.toPrimitive]()
    logger.debug(`intervalId: ${intervalId}`)
    return intervalId
  }

  return wrapper
}
