import { parseArgs } from 'node:util'
import { fetchDataWithAdapter, loadAdapters } from '../worker/utils'

async function main() {
  const adapterId: string = loadArgs()
  const adapters = await loadAdapters({ postprocess: true })
  const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms * 1000))
  let roundCount = 1
  while (true) {
    const price = await fetchDataWithAdapter(
      adapters[adapterId].feeds,
      adapters[adapterId].name,
      roundCount++
    )
    console.log({ roundCount: roundCount - 1, price, time: new Date() })
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
