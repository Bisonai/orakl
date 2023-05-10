import path from 'node:path'
import os from 'node:os'
import pino from 'pino'
import { LOG_LEVEL, LOG_DIR } from './settings'

export function buildLogger(name: string) {
  const transport = pino.transport({
    targets: [
      { target: 'pino-pretty', level: LOG_LEVEL, options: { destination: 1 } },
      {
        target: 'pino/file',
        level: LOG_LEVEL,
        options: {
          destination: path.join(LOG_DIR, `orakl-${os.hostname()}-${name}.log`)
        }
      }
    ]
  })

  const logger = pino(transport)
  logger.level = LOG_LEVEL

  return logger
}

/**
 * Logger created with this function is expected to be used within
 * tests, however, because of its properties, it can be used also at
 * other places, but it is not recommended. Logs generated by this
 * logger will be send to standard output, and formatted for a better
 * readability.
 */
export function buildMockLogger(name: string) {
  const transport = pino.transport({
    targets: [{ target: 'pino-pretty', level: LOG_LEVEL, options: { destination: 1 } }]
  })

  const logger = pino(transport)
  return logger
}
