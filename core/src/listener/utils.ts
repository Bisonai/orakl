import { IListenerConfig } from '../types'

export function validateListenerConfig(config: IListenerConfig[]): boolean {
  const properties = ['address', 'eventName']

  for (const c of config) {
    const propertyExist = properties.map((p) => (c[p] ? true : false))
    const allPropertiesExist = propertyExist.every((i) => i)
    if (!allPropertiesExist) {
      return false
    }
  }

  return true
}
