import * as Fs from 'node:fs/promises'
import * as Path from 'node:path'
import { Worker } from 'bullmq'
import { got, Options } from 'got'
import { IcnError, IcnErrorCode } from './errors.js'
import { IAdapter } from './types.js'
import { buildAdapterRootDir } from './utils.js'
import { buildBullMqConnection, buildQueueName, loadJson } from './utils.js'
import { parseReducer, mulReducer, reducerMapping } from './reducer.js'

function prepareFeed(adapter) {
  console.log(adapter)

  return adapter.feeds.map((f) => {
    const reducers = f.reducers?.map((r) => {
      // TODO check if exists
      return reducerMapping[r.function](r.args)
    })

    const adapterId = 'kfew' // adapter.adapter_id
    return {
      [adapterId]: {
        url: f.url,
        headers: f.headers,
        method: f.method,
        reducers
      }
    }
  })
}

async function loadAdapters() {
  const adapterRootDir = buildAdapterRootDir()
  const adapterPaths = await Fs.readdir(adapterRootDir)

  const allRawAdapters = await Promise.all(
    adapterPaths.map(async (ap) => validateAdapter(await loadJson(Path.join(adapterRootDir, ap))))
  )
  const activeRawAdapters = allRawAdapters.filter((a) => a.active)

  const adapters = activeRawAdapters.map((a) => prepareFeed(a))
  console.log('adapters', adapters)

  return Object.assign({}, ...adapters)
}

function validateAdapter(adapter): IAdapter {
  // TODO extract properties from Interface
  const requiredProperties = ['active', 'name', 'job_type', 'adapter_id', 'oracle', 'feeds']
  // TODO show where is the error
  const hasProperty = requiredProperties.map((p) => adapter.hasOwnProperty(p))
  const isValid = hasProperty.every((x) => x)

  if (isValid) {
    return adapter as IAdapter
  } else {
    throw new IcnError(IcnErrorCode.InvalidOperator)
  }
}

async function fetch() {}

async function main() {
  const adapters = await loadAdapters()
  console.log('adapters', adapters)

  const worker = new Worker(
    buildQueueName(),
    async (job) => {
      console.log(job.data)

      const adapter = adapters['efbdab54419511edb8780242ac120002'] // FIXME
      console.log('current adapter', adapter)

      const data = await got(adapter.url, {
        headers: {
          ...adapter.headers,
          ...adapter.method
        }
      }).json()
      console.log(data)

      let tmp = data
      for (const r of adapter.reducers) {
        tmp = r(tmp)
      }
      console.log(tmp)

      // TODO global reducer
    },
    buildBullMqConnection()
  )
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
