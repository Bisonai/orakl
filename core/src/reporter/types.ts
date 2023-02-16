import { Logger } from 'pino'

export interface IReporters {
  [index: string]: (_logger: Logger) => Promise<void>
}

export interface ISubmissionInfo {
  toSubmitRoundId: number
  submittedRoundId: number
  submitter: string | null
}
