import { parseArgs } from 'node:util'
import { buildLogger } from '../logger'
import { aggregatorWorker } from './aggregator'
import { vrfWorker } from './vrf'
import { worker as requestResponseWorker } from './request-response'
import { predefinedFeedWorker } from './predefined-feed'
import { healthCheck } from '../health-checker'
import { hookConsoleError } from '../utils'

const WORKERS = {
  AGGREGATOR: aggregatorWorker,
  VRF: vrfWorker,
  REQUEST_RESPONSE: requestResponseWorker,
  PREDEFINED_FEED: predefinedFeedWorker
}

const LOGGER = buildLogger('worker')

async function main() {
  hookConsoleError(LOGGER)
  const worker = loadArgs()
  WORKERS[worker](LOGGER)
  healthCheck()
}

function loadArgs() {
  const {
    values: { worker }
  } = parseArgs({
    options: {
      worker: {
        type: 'string'
      }
    }
  })

  if (!worker) {
    throw Error('Missing --worker argument.')
  }

  if (!Object.keys(WORKERS).includes(worker)) {
    throw Error(`${worker} is not supported worker.`)
  }

  return worker
}

main().catch((e) => {
  LOGGER.error(e)
  process.exitCode = 1
})
