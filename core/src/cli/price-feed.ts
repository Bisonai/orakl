import { parseArgs } from 'node:util'
import { fetchDataWithAdapter, loadAdapters } from '../worker/utils'

async function main() {
  const adapterId: string = loadArgs()
  const adapters = await loadAdapters({ postprocess: true })
  fetchDataWithAdapter(adapters[adapterId])
}

function loadArgs(): string {
  const {
    values: { adapterId }
  } = parseArgs({
    options: {
      adapterId: {
        type: 'string'
      }
    }
  })

  if (!adapterId) {
    throw Error('Missing --adapterId argument')
  }

  return adapterId
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
