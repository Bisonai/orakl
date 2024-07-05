import { Logger } from 'pino'
import { OraklError, OraklErrorCode } from '../errors'
import { IListenerConfig, IListenerGroupConfig, IListenerRawConfig } from '../types'
import { isAddressValid } from '../utils'

const FILE_NAME = import.meta.url

/**
 * Group listener raw configurations based on `service` property.
 * Listener Raw Configuration
 *  [
 *    {
 *      address: '0x123',
 *      eventName: 'RandomWordsRequested',
 *      chain: 'localhost',
 *      service: 'VRF'
 *    },
 *    {
 *      address: '0x456',
 *      eventName: 'RandomWordsRequested',
 *      chain: 'localhost',
 *      service: 'VRF'
 *    },
 *    {
 *      address: '0x000',
 *      eventName: 'NewRound',
 *      chain: 'localhost',
 *      service: 'Aggregator'
 *    }
 *  ]
 *
 * Listener Group Configuration
 *  {
 *    'VRF': [
 *      {
 *        'address': '0x123',
 *        'eventName': 'RandomWordsRequested',
 *        'chain': 'localhost'
 *      },
 *      {
 *        'address': '0x456',
 *        'eventName': 'RandomWordsRequested',
 *        'chain': 'localhost'
 *      }
 *    ],
 *    'Aggregator': [
 *      {
 *        'address': '0x000',
 *        'eventName': 'NewRound',
 *        'chain': 'localhost'
 *      }
 *    ]
 *  }
 *
 * @param {IListenerRawConfig[]} list of listener raw configurations
 * @return {IListenerGroupConfig} grouped raw listener configurations based on `service` property
 */
export function groupListeners({
  listenersRawConfig,
}: {
  listenersRawConfig: IListenerRawConfig[]
}): IListenerGroupConfig {
  const postprocessed = listenersRawConfig.reduce((groups, item) => {
    const group = groups[item.service] || []
    group.push(item)
    groups[item.service] = group
    return groups
  }, {})

  Object.keys(postprocessed).forEach((serviceName) => {
    return postprocessed[serviceName].map((listener) => {
      delete listener.service
      return listener
    })
  })

  return postprocessed
}

/**
 * Check whether every listener within a listener config contains
 * required properties: `address` and `eventName`.
 *
 * @param {IListenerConfig[]} listener configuration used for launching listeners
 * @param {pino.Logger?}
 * @return {boolean} true when the given listener configuration is valid, otherwise false
 */
export function validateListenerConfig(config: IListenerConfig[], logger?: Logger): boolean {
  const requiredProperties = ['address', 'eventName']

  for (const c of config) {
    const propertyExist = requiredProperties.map((p) => (c[p] ? true : false))
    const allPropertiesExist = propertyExist.every((i) => i)
    if (!allPropertiesExist) {
      logger?.error({ name: 'validateListenerConfig', file: FILE_NAME, ...c }, 'config')
      return false
    }
    if (!isAddressValid(c.address)) {
      logger?.error(
        { name: 'validateListenerConfig', file: FILE_NAME, address: c.address },
        'invalid address',
      )
      return false
    }
  }

  return true
}

export function postprocessListeners({
  listenersRawConfig,
  service,
  chain,
  logger,
}: {
  listenersRawConfig: IListenerRawConfig[]
  service: string
  chain: string
  logger?: Logger
}): IListenerGroupConfig {
  if (listenersRawConfig.length == 0) {
    throw new OraklError(
      OraklErrorCode.NoListenerFoundGivenRequirements,
      `service: [${service}], chain: [${chain}]`,
    )
  }
  const listenersConfig = groupListeners({ listenersRawConfig })
  const isValid = Object.keys(listenersConfig).map((k) =>
    validateListenerConfig(listenersConfig[k], logger),
  )

  if (!isValid.every((t) => t)) {
    throw new OraklError(OraklErrorCode.InvalidListenerConfig)
  }

  logger?.info(
    { name: 'postprocessListeners', file: FILE_NAME, ...listenersConfig },
    'listenersConfig',
  )

  return listenersConfig
}
