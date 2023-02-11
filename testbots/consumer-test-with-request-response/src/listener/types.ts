import { Logger } from 'pino'
import { IListenerConfig } from '../types'

export interface IListeners {
  [index: string]: (config: IListenerConfig[], logger: Logger) => void
}
