import { parseArgs } from 'node:util'
import { computeDataHash } from './orakl-cli/src/utils'
import { loadJson } from '../utils'
import { IAggregator } from '../types'

async function main() {
  const { aggregatorPaths, verify } = loadArgs()
  const aggregators: IAggregator[] = []

  if (aggregatorPaths.length) {
    for (const ap of aggregatorPaths) {
      aggregators.push(await loadJson(ap))
    }
  }

  for (const data of aggregators) {
    await computeDataHash({ data, verify })
  }
}

function loadArgs() {
  const {
    values: { verify },
    positionals: aggregatorPaths
  } = parseArgs({
    options: {
      verify: {
        type: 'boolean',
        default: false
      }
    },
    allowPositionals: true
  })

  return { aggregatorPaths, verify }
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
