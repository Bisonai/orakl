import { Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import { PROVIDER_URL, BULLMQ_CONNECTION, LISTENER_DELAY } from '../settings'
import { IListenerConfig } from '../types'
import { PubSubStop } from './pub-sub-stop'

export function listen({
  queueName,
  processEventFn,
  abi,
  redisClient,
  pubsub,
  logger
}: {
  queueName: string
  processEventFn
  abi
  redisClient
  pubsub: PubSubStop
  logger: Logger
}) {
  async function wrapper(listener: IListenerConfig) {
    const provider = new ethers.providers.JsonRpcProvider(PROVIDER_URL)
    const contract = new ethers.Contract(listener.address, abi, provider)
    const iface = new ethers.utils.Interface(abi)
    const queue = new Queue(queueName, BULLMQ_CONNECTION)
    const processEvent = processEventFn(iface, queue, logger)
    const listenerRedisKey = `listener:${listener.id}`

    let observedBlock = (await redisClient.get(listenerRedisKey)) || 0

    const listenerId = setInterval(async () => {
      try {
        const latestBlock = await provider.getBlockNumber()
        console.log(`latest: ${latestBlock}, observedBlock: ${observedBlock}`)
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
        console.error(e)
        logger.error({ name: 'listen:wrapper' }, e)
      }
    }, LISTENER_DELAY)
    pubsub.setupSubscriber(listenerId, listener.id)
  }

  return wrapper
}
