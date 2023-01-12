import { loadAdapters, buildReducer } from '../worker/utils'
import { pipe } from '../utils'
import got from 'got'
import { localAggregatorFn } from '../settings'
import { IcnError, IcnErrorCode } from '../errors'
import { parseArgs } from 'node:util'

function checkPriceFeedFormat(priceFeed) {
  if (!priceFeed) {
    // check if priceFeed is null,undefined,NaN,"",0,false
    throw new IcnError(IcnErrorCode.InvalidPriceFeed)
  } else if (!Number.isInteger(priceFeed)) {
    // check if priceFeed is not Integer
    throw new IcnError(IcnErrorCode.InvalidPriceFeedFormat)
  }
}

export async function fetchPriceFeed(adapterId: string) {
  const adapters = await loadAdapters()
  try {
    const allResults = await Promise.all(
      adapters[adapterId].map(async (adapter) => {
        const options = {
          method: adapter.method,
          headers: adapter.headers
        }
        try {
          const rawData = await got(adapter.url, options).json()
          const reducers = buildReducer(adapter.reducers)
          const priceFeed = pipe(...reducers)(rawData)

          checkPriceFeedFormat(priceFeed)

          console.log('src/cli/predefinedFeedJob:url:', adapter.url)
          console.log('src/cli/predefinedFeedJob:PriceFeed:', priceFeed)
          return priceFeed
        } catch (e) {
          console.error(e)
          return
        }
      })
    )
    const res = localAggregatorFn(allResults)
    console.log('src/cli/predefinedFeedJob:AllResults:', allResults)
    console.log('src/cli/predefinedFeedJob:res', res)
    return res
  } catch (e) {
    console.error(e)
  }
}

function main() {
  const adapterId: string = loadArgs()
  return fetchPriceFeed(adapterId)
}

function loadArgs() {
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
