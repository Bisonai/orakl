import { parseArgs } from 'node:util'
import { computeDataHash } from './orakl-cli/src/utils'
import { loadJson } from '../utils'
import { IAdapter } from '../types'

async function main() {
  const { adapterPaths, verify } = loadArgs()
  const adapters: IAdapter[] = []

  if (adapterPaths.length) {
    for (const ap of adapterPaths) {
      adapters.push(await loadJson(ap))
    }
  }

  for (const data of adapters) {
    await computeDataHash({ data, verify })
  }
}

function loadArgs() {
  const {
    values: { verify },
    positionals: adapterPaths
  } = parseArgs({
    options: {
      verify: {
        type: 'boolean',
        default: false
      }
    },
    allowPositionals: true
  })

  return { adapterPaths, verify }
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
