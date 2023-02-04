import pino from 'pino'
import { LOG_LEVEL, LOG_FILE_PATH } from './settings'

export function buildLogger(name: string) {
  const transport = pino.transport({
    targets: [
      { target: 'pino-pretty', level: LOG_LEVEL, options: { destination: 1 } },
      {
        target: 'pino/file',
        level: LOG_LEVEL,
        options: {
          destination: `orakl-${name}.log`
        }
      }
    ]
  })

  const logger = pino(transport)
  logger.level = LOG_LEVEL

  return logger
}
