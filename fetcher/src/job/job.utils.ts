import { buildReducer, checkDataFormat, pipe, REDUCER_MAPPING } from '@bisonai/orakl-util'
import { Logger } from '@nestjs/common'
import axios from 'axios'
import { Contract, JsonRpcProvider } from 'ethers'
import { abis as uniswapPoolAbis } from '../abis/pool'
import { CYPRESS_PROVIDER_URL, ETHEREUM_PROVIDER_URL, FETCH_TIMEOUT } from '../settings'
import { LOCAL_AGGREGATOR_FN } from './job.aggregator'
import { FetcherError, FetcherErrorCode } from './job.errors'
import { IAdapter, IFetchedData, IProxy } from './job.types'

const CYPRESS_CHAIN_ID = 8217
const ETHEREUM_CHAIN_ID = 1

export function buildUrl(host: string, path: string) {
  const url = [host, path].join('/')
  return url.replace(/([^:]\/)\/+/g, '$1')
}

function isProxyDefined(adapter) {
  return (
    adapter.proxy !== undefined &&
    adapter.proxy.protocol !== undefined &&
    adapter.proxy.host !== undefined &&
    adapter.proxy.port !== undefined
  )
}

const INVALID_DATA = -1
async function fetchCall(url: string, options, logger) {
  try {
    return (await axios.get(url, options)).data
  } catch (e) {
    logger.error(`Error in fetching data from ${url}: ${e.message}`)
    logger.error(e)
    return INVALID_DATA
  }
}

async function fetchRawDataWithProxy(adapter, logger) {
  return fetchCall(
    adapter.url,
    {
      method: adapter.method,
      headers: adapter.headers,
      timeout: FETCH_TIMEOUT,
      proxy: {
        protocol: adapter.proxy.protocol,
        host: adapter.proxy.host,
        port: adapter.proxy.port,
      },
    },
    logger,
  )
}

async function fetchRawDataWithoutProxy(adapter, logger) {
  return fetchCall(
    adapter.url,
    {
      method: adapter.method,
      headers: adapter.headers,
      timeout: FETCH_TIMEOUT,
    },
    logger,
  )
}

/**
 * Fetch data from data sources defined in `adapter`.
 *
 * @param {} adapter Single data adapter to define which data to fetch.
 * @param {} NestJs logger
 * @return {number} aggregatedresults
 */
export async function fetchData(adapterList, decimals, logger) {
  const data = await Promise.allSettled(
    adapterList.map(async (adapter) => {
      let rawDatum = INVALID_DATA
      if (!adapter.type) {
        if (isProxyDefined(adapter)) {
          rawDatum = await fetchRawDataWithProxy(adapter, logger)
        }
        if (rawDatum === INVALID_DATA) {
          rawDatum = await fetchRawDataWithoutProxy(adapter, logger)
          if (rawDatum === INVALID_DATA) {
            throw new Error(`Error in fetching data`)
          }
        }
      }

      try {
        // FIXME Build reducers just once and use. Currently, can't
        // be passed to queue, therefore has to be recreated before
        // every fetch.
        let datum
        if (adapter.type == 'UniswapPool') {
          datum = await extractUniswapPrice(adapter, decimals)
        } else {
          const reducers = buildReducer(REDUCER_MAPPING, adapter.reducers)
          datum = pipe(...reducers)(rawDatum)
        }
        checkDataFormat(datum)
        return { id: adapter.id, value: datum }
      } catch (e) {
        logger.error(`Fetching with proxy ${adapter.proxy.host} failed in ${adapter.name}`)
        logger.error(e)
        throw e
      }
    }),
  )

  return data.flatMap((D) => (D.status == 'fulfilled' ? [D.value] : []))
}

export function aggregateData(data: IFetchedData[]): number {
  const aggregate = LOCAL_AGGREGATOR_FN(data.map((d) => d.value))
  return aggregate
}

function validateAdapter(adapter): IAdapter {
  // TODO extract properties from Interface
  const requiredProperties = ['id', 'active', 'name', 'jobType', 'decimals', 'feeds']
  // TODO show where is the error
  const hasProperty = requiredProperties.map((p) =>
    Object.prototype.hasOwnProperty.call(adapter, p),
  )
  const isValid = hasProperty.every((x) => x)

  if (isValid) {
    return adapter as IAdapter
  } else {
    throw new FetcherError(FetcherErrorCode.InvalidAdapter)
  }
}

function selectProxyFn(proxies: IProxy[]) {
  let latestId = 0
  const proxySourceMap: { [host: string]: number } = {}

  function wrapper(url: string): IProxy {
    const source = new URL(url).host
    const proxySize = proxies.length
    if (proxySize == 0) {
      return { protocol: undefined, host: undefined, port: undefined }
    }

    if (source in proxySourceMap) {
      proxySourceMap[source] = (proxySourceMap[source] + 1) % proxySize
    } else {
      proxySourceMap[source] = latestId
      latestId = (latestId + 1) % proxySize
    }
    return proxies[proxySourceMap[source]]
  }
  return wrapper
}

export function extractFeeds(
  adapter,
  aggregatorId: bigint,
  aggregatorHash: string,
  threshold: number,
  absoluteThreshold: number,
  address: string,
  proxies: IProxy[],
  logger: Logger,
) {
  const adapterHash = adapter.adapterHash
  const proxySelector = selectProxyFn(proxies)
  const feeds = adapter.feeds.map((f) => {
    let proxy: IProxy
    try {
      if (!f.definition.location) {
        proxy = proxySelector(f.definition.url)
      } else {
        const availableProxies = proxies.filter(
          (item) => item.location && item.location === f.definition.location,
        )
        if (availableProxies.length == 0) {
          throw `no proxies available for location:${f.definition.location}`
        }
        const randomIndex = Math.floor(Math.random() * availableProxies.length)
        proxy = availableProxies[randomIndex]
      }
    } catch (e) {
      logger.error('Assigning proxy has failed')
      logger.error(e)
    }

    return {
      id: f.id,
      name: f.name,
      proxy,
      ...f.definition,
    }
  })

  return {
    [adapterHash]: {
      aggregatorId: aggregatorId,
      aggregatorHash: aggregatorHash,
      name: adapter.name,
      decimals: adapter.decimals,
      threshold,
      absoluteThreshold,
      address,
      feeds,
    },
  }
}

/**
 * Test whether the current submission deviates from the last
 * submission more than given threshold or absolute threshold. If yes,
 * return `true`, otherwise `false`.
 *
 * @param {number} latest submission value
 * @param {number} current submission value
 * @param {number} threshold configuration
 * @param {number} absolute threshold configuration
 * @return {boolean}
 */
export function shouldReport(
  latestSubmission: number,
  submission: number,
  decimals: number,
  threshold: number,
  absoluteThreshold: number,
): boolean {
  if (latestSubmission && submission) {
    const denominator = Math.pow(10, decimals)
    const latestSubmissionReal = latestSubmission / denominator
    const submissionReal = submission / denominator

    const range = latestSubmissionReal * threshold
    const l = latestSubmissionReal - range
    const r = latestSubmissionReal + range
    return submissionReal < l || submissionReal > r
  } else if (!latestSubmission && submission) {
    // latestSubmission hit zero
    return submission > absoluteThreshold
  } else {
    // Something strange happened, don't report!
    return false
  }
}

export async function extractUniswapPrice(adapter, decimals) {
  const provider = await providerByChain(Number(adapter.chainId))
  const poolContract = new Contract(adapter.address, uniswapPoolAbis, provider)
  const rawData = await poolContract.slot0()
  const datum = sqrtPriceX96ToTokenPrice(
    BigInt(rawData[0]),
    adapter.token0Decimals,
    adapter.token1Decimals,
  )
  if (adapter.reciprocal) {
    if (datum === 0) {
      throw new Error(`Division by zero err in extractUniswapPrice`)
    }
    return Math.round((1 / datum) * 10 ** decimals)
  }
  return Math.round(datum * 10 ** decimals)
}

export async function providerByChain(chainId: number) {
  if (chainId == CYPRESS_CHAIN_ID) {
    return new JsonRpcProvider(CYPRESS_PROVIDER_URL)
  } else if (chainId == ETHEREUM_CHAIN_ID) {
    return new JsonRpcProvider(ETHEREUM_PROVIDER_URL)
  } else {
    throw new Error(`Invalid chain id`)
  }
}

function sqrtPriceX96ToTokenPrice(
  sqrtPriceX96: bigint,
  decimal0: number,
  decimal1: number,
): number {
  return (
    Math.pow(Number(sqrtPriceX96) / Math.pow(2, 96), 2) /
    (Math.pow(10, decimal1) / Math.pow(10, decimal0))
  )
}
