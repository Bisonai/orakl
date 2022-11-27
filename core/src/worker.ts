import { Worker } from 'bullmq'
import { buildBullMqConnection, buildQueueName, loadJson } from './utils.js'
import { got } from 'got'
import { IcnError, IcnErrorCode } from './errors.js'
import { IAdapter } from './types.js'
import { buildAdapterRootDir } from './utils.js'
import * as Fs from 'node:fs/promises'
import * as Path from 'node:path'

async function loadAdapters() {
  const adapterRootDir = buildAdapterRootDir()
  const adapterPaths = await Fs.readdir(adapterRootDir)

  const allAdapters = await Promise.all(
    adapterPaths.map(async (ap) => validateAdapter(await loadJson(Path.join(adapterRootDir, ap))))
  )
  return allAdapters.filter((a) => a.active)
}

function validateAdapter(adapter): IAdapter {
  // TODO show where is the error
  const requiredProperties = ['active', 'name', 'job_type', 'adapter_id', 'oracle', 'feed']
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

      const data = await got
        .post('https://httpbin.org/anything', {
          json: {
            hello: 'world'
          }
        })
        .json()
      console.log(data)
    },
    buildBullMqConnection()
  )
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
