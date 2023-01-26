import { parseArgs } from 'node:util'
import { aggregatorReporter } from './aggregator'
import { vrfReporter } from './vrf'
import { anyApiReporter } from './any-api'
import { healthCheck } from '../health-checker'
import { hookConsoleError } from '../utils'

const REPORTERS = {
  AGGREGATOR: aggregatorReporter,
  VRF: vrfReporter,
  ANY_API: anyApiReporter
}

async function main() {
  hookConsoleError()
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
