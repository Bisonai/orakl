import * as Fs from 'node:fs/promises'
import * as Path from 'node:path'
import { Worker, Queue } from 'bullmq'
import { got, Options } from 'got'
import { IcnError, IcnErrorCode } from './errors.js'
import { IAdapter } from './types.js'
import { buildBullMqConnection, loadJson, pipe } from './utils.js'
import { buildAdapterRootDir } from './utils.js'
import { reducerMapping } from './reducer.js'
import { localAggregatorFn, workerRequestQueueName, reporterRequestQueueName } from './settings.js'

function extractFeeds(adapter) {
  const adapterId = adapter.adapter_id
  const feeds = adapter.feeds.map((f) => {
    const reducers = f.reducers?.map((r) => {
      // TODO check if exists
      return reducerMapping[r.function](r.args)
    })

    return {
      url: f.url,
      headers: f.headers,
      method: f.method,
      reducers
    }
  })

  return { [adapterId]: feeds }
}

async function loadAdapters() {
  const adapterRootDir = buildAdapterRootDir()
  const adapterPaths = await Fs.readdir(adapterRootDir)

  const allRawAdapters = await Promise.all(
    adapterPaths.map(async (ap) => validateAdapter(await loadJson(Path.join(adapterRootDir, ap))))
  )
  const activeRawAdapters = allRawAdapters.filter((a) => a.active)
  return activeRawAdapters.map((a) => extractFeeds(a))
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

async function main() {
  const adapters = (await loadAdapters())[0] // FIXME take all adapters
  console.log('adapters', adapters)

  // FIXME take adapterId from job.data (information emitted by on-chain event)
  const adapterId = 'efbdab54419511edb8780242ac120002'

  const queue = new Queue(reporterRequestQueueName, buildBullMqConnection())

  // TODO if job not finished, return job in queue

  const worker = new Worker(
    workerRequestQueueName,
    async (job) => {
      console.log('request', job.data)

      if (true) {
        const {
          requestId,
          nonce,
          callbackAddress,
          callbackFunctionId,
          _data: dataRequest
        } = job.data
        const url = Buffer.from(dataRequest.substring(2), 'hex').toString().substring(6)
        console.log(url)
        const rawData: any = await got(url).json()
        console.log(rawData)
        console.log(rawData['DISPLAY']['ETH']['USD'])
        const data = rawData['RAW']['ETH']['USD']['PRICE'] // FIXME make it more general
        console.log(data)
        await queue.add('reporter', {
          requestId,
          callbackAddress,
          callbackFunctionId,
          data
        })
      }

      // const results = await Promise.all(
      //   adapters[adapterId].map(async (adapter) => {
      //     console.log('current adapter', adapter)
      //
      //     const options = {
      //       method: adapter.method,
      //       headers: adapter.headers
      //     }
      //
      //     try {
      //       const rawData = await got(adapter.url, options).json()
      //       return pipe(...adapter.reducers)(rawData)
      //       // console.log(`data ${data}`)
      //     } catch (e) {
      //       // FIXME
      //       console.log(e)
      //     }
      //   })
      // )
      // console.log(results)

      // FIXME single node aggregation of multiple results
      // FIXME check if aggregator is defined and if exists
      // try {
      //   const aggregatedResult = localAggregatorFn(...results)
      //   console.log(aggregatedResult)
      // } catch (e) {
      //   console.log(e)
      // }
    },
    buildBullMqConnection()
  )
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
