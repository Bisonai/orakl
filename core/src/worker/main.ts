import { parseArgs } from 'node:util'
import { buildLogger } from '../logger'
import { worker as aggregatorWorker } from './aggregator'
import { worker as vrfWorker } from './vrf'
import { worker as requestResponseWorker } from './request-response'
import { launchHealthCheck } from '../health-check'
import { hookConsoleError } from '../utils'

const WORKERS = {
  AGGREGATOR: aggregatorWorker,
  VRF: vrfWorker,
  REQUEST_RESPONSE: requestResponseWorker
}

const LOGGER = buildLogger('worker')

async function main() {
  hookConsoleError(LOGGER)
  const worker = loadArgs()
  WORKERS[worker](LOGGER)
  launchHealthCheck()
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
