import { Logger } from 'pino'
import { IListenerConfig } from '../types'

interface IListener {
  queueName: string
  fn: (queueName: string, config: IListenerConfig[], logger: Logger) => void
}

export interface IListeners {
  [index: string]: IListener
}
