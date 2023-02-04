import { parseArgs } from 'node:util'
import { buildLogger } from '../logger'
import { aggregatorReporter } from './aggregator'
import { vrfReporter } from './vrf'
import { reporter as requestResponseReporter } from './request-response'
import { healthCheck } from '../health-checker'
import { hookConsoleError } from '../utils'

const REPORTERS = {
  AGGREGATOR: aggregatorReporter,
  VRF: vrfReporter,
  REQUEST_RESPONSE: requestResponseReporter
}

const LOGGER = buildLogger('reporter')

async function main() {
  hookConsoleError(LOGGER)
  const reporter = loadArgs()
  REPORTERS[reporter](LOGGER)
  healthCheck()
}

function loadArgs() {
  const {
    values: { reporter }
  } = parseArgs({
    options: {
      reporter: {
        type: 'string'
      }
    }
  })

  if (!reporter) {
    throw Error('Missing --reporter argument.')
  }

  if (!Object.keys(REPORTERS).includes(reporter)) {
    throw Error(`${reporter} is not supported reporter.`)
  }

  return reporter
}

main().catch((error) => {
  LOGGER.error(error)
  process.exitCode = 1
})
