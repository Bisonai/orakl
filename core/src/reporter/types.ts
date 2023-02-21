import { Logger } from 'pino'

export interface IReporters {
  [index: string]: (_logger: Logger) => Promise<void>
}
