import { Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import { VRFCoordinator__factory } from '@bisonai/orakl-contracts'
import { IListenerConfig, IRandomWordsRequested, IVrfListenerWorker } from '../types'
import { WORKER_VRF_QUEUE_NAME, CHAIN } from '../settings'
import { getVrfConfig } from '../api'
import { listen } from './listener'

const FILE_NAME = import.meta.url
const { keyHash: KEY_HASH } = await getVrfConfig({ chain: CHAIN })
import { watchman } from './watchman'

export async function buildListener(config: IListenerConfig[], redisClient, logger: Logger) {
  const queueName = WORKER_VRF_QUEUE_NAME

  const configName = 'listener-vrf-config' // TODO deployment-name
  const service = 'VRF'

  // Previously stored listener config is ignored,
  // and replaced with the latest state of Orakl Network.
  await redisClient.set(configName, JSON.stringify(config))

  const listenFn = listen({
    queueName,
    processEventFn: processEvent,
    abi: VRFCoordinator__factory.abi,
    redisClient,
    logger
  })

  for (const listener of config) {
    listenFn(listener)
  }

  watchman({ redisClient, listenFn, service, chain: CHAIN })
}

function processEvent(iface: ethers.utils.Interface, queue: Queue, _logger: Logger) {
  const logger = _logger.child({ name: 'processEvent', file: FILE_NAME })

  async function wrapper(log) {
    const eventData = iface.parseLog(log).args as unknown as IRandomWordsRequested
    logger.debug(eventData, 'eventData')

    if (eventData.keyHash != KEY_HASH) {
      logger.info(`Ignore event with keyhash [${eventData.keyHash}]`)
    } else {
      const data: IVrfListenerWorker = {
        callbackAddress: log.address,
        blockNum: log.blockNumber,
        blockHash: log.blockHash,
        requestId: eventData.requestId.toString(),
        seed: eventData.preSeed.toString(),
        accId: eventData.accId.toString(),
        callbackGasLimit: eventData.callbackGasLimit,
        numWords: eventData.numWords,
        sender: eventData.sender,
        isDirectPayment: eventData.isDirectPayment
      }
      logger.debug(data, 'data')

      await queue.add('vrf', data, {
        jobId: data.requestId,
        removeOnComplete: {
          age: 1_800
        }
      })
    }
  }

  return wrapper
}
