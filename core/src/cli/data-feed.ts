import { parseArgs } from 'node:util'
import { fetchDataWithAdapter, loadAdapters } from '../worker/utils'
import { STORE_ADAPTER_FETCH_RESULT } from '../settings'

async function main() {
  const adapterId: string = loadArgs()
  const adapters = await loadAdapters({ postprocess: true })
  let round = 1
  while (STORE_ADAPTER_FETCH_RESULT) {
    const price = await fetchDataWithAdapter(adapters[adapterId].feeds, round++)
    const now = new Date()
    console.log(`Round: ${round - 1}, Price: ${price}, Time:${now}\n\n`)

    const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms * 1000))
    await sleep(1)
  }
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
