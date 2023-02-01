import { parseArgs } from 'node:util'
import { aggregatorReporter } from './aggregator'
import { vrfReporter } from './vrf'
import { reporter as requestResponseReporter } from './request-response'
import { healthCheck } from '../health-checker'

const REPORTERS = {
  AGGREGATOR: aggregatorReporter,
  VRF: vrfReporter,
  REQUEST_RESPONSE: requestResponseReporter
}

async function main() {
  const reporter = loadArgs()
  REPORTERS[reporter]()
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
  console.error(error)
  process.exitCode = 1
})
