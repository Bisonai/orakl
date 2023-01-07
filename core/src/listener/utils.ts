import { IListenerConfig } from '../types'

export function validateListenerConfig(config: IListenerConfig): boolean {
  const properties = ['address', 'eventName', 'factoryName']
  const propertyExist = properties.map((p) => (config[p] ? true : false))
  const allPropertiesExist = propertyExist.every((i) => i)
  return allPropertiesExist
}
