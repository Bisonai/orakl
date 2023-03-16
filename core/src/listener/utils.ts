import { IListenerRawConfig, IListenerConfig, IListenerGroupConfig } from '../types'
import { Logger } from 'pino'

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
export function postprocessListeners({
  listenersRawConfig
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
  }

  return true
}
