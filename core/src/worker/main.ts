import { parseArgs } from 'node:util'
import { aggregatorWorker } from './aggregator'
import { vrfWorker } from './vrf'
import { anyApiWorker } from './any-api'
import { predefinedFeedWorker } from './predefined-feed'
import { healthCheck } from '../health-checker'
import { hookConsoleError } from '../utils'

const WORKERS = {
  AGGREGATOR: aggregatorWorker,
  VRF: vrfWorker,
  ANY_API: anyApiWorker,
  PREDEFINED_FEED: predefinedFeedWorker
}

async function main() {
  hookConsoleError()
  const worker = loadArgs()
  WORKERS[worker]()
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

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
