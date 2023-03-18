import { Queue } from 'bullmq'
import { Contract, ethers } from 'ethers'
import { Logger } from 'pino'
import { PROVIDER_URL, BULLMQ_CONNECTION, LISTENER_DELAY } from '../settings'
import { IListenerBlock, IListenerConfig } from '../types'

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
            // logger.debug({ name: 'Event:filter' }, `${events}`)
            events.forEach(processEvent)
          }
        }

        observedBlock = latestBlock
        await redisClient.set(listenerRedisKey, observedBlock)
      } catch (e) {
        console.error(e)
        // logger.error({ name: 'Event:filter' }, e)
      }
    }, LISTENER_DELAY)

    // Subcription to stop listener loop
    const subscriber = redisClient.duplicate()
    await subscriber.connect()
    const channelName = `listener:stop:${listener.id}` // FIXME
    const stopListener = async (message, channel) => {
      clearInterval(listenerId)
      await subscriber.unsubscribe(channelName)
    }
    await subscriber.subscribe(channelName, stopListener)
  }

  return wrapper
}
