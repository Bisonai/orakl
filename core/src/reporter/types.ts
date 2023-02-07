import { Logger } from 'pino'
import { IListenerConfig } from '../types'

export interface IReporters {
  [index: string]: (_logger: Logger) => Promise<void>
}
