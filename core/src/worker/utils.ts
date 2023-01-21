import { reducerMapping } from './reducer'
import { IcnError, IcnErrorCode } from '../errors'
import { pipe } from '../utils'
import { IAdapter, IAggregator } from '../types'
import { getAdapters, getAggregators, localAggregatorFn, DB, CHAIN } from '../settings'
import axios from 'axios'

export async function loadAdapters({ postprocess }: { postprocess?: boolean }) {
  const rawAdapters = await getAdapters(DB, CHAIN)
  const validatedRawAdapters = rawAdapters.map((a) => validateAdapter(JSON.parse(a.data)))

  if (!postprocess) {
    return validatedRawAdapters
  }

  const activeRawAdapters = validatedRawAdapters.filter((a) => a.active)
  return Object.assign({}, ...activeRawAdapters.map((a) => extractFeeds(a)))
}

export async function loadAggregators({ postprocess }: { postprocess?: boolean }) {
  const rawAggregators = await getAggregators(DB, CHAIN)
  const validatedRawAggregators = rawAggregators.map((a) => validateAggregator(JSON.parse(a.data)))

  if (!postprocess) {
    return validatedRawAggregators
  }

  const activeRawAggregators = validatedRawAggregators.filter((a) => a.active)
  return Object.assign({}, ...activeRawAggregators.map((a) => extractAggregators(a)))
}

export function mergeAggregatorsAdapters(aggregators, adapters) {
  // FIXME use mapping instead
  // TODO replace any
  /* eslint-disable @typescript-eslint/no-explicit-any */
  const aggregatorsWithAdapters: any = []

  for (const agAddress in aggregators) {
    const ag = aggregators[agAddress]
    if (ag) {
      ag['adapter'] = adapters[ag.adapterId]
      aggregatorsWithAdapters.push({ [agAddress]: ag })
    } else {
      throw new IcnError(IcnErrorCode.MissingAdapter)
    }
  }

  return Object.assign({}, ...aggregatorsWithAdapters)
}

/**
 * Fetch data from API endpoints defined in `adapter`.
 *
 * @param {number} adapter Single data adapter to define which data to fetch.
 * @return {number} aggregatedresults
 * @exception {InvalidPriceFeed} raised when there is at least one undefined data point
 */
export async function fetchDataWithAdapter(adapter) {
  const allResults = await Promise.all(
    adapter.map(async (a) => {
      const options = {
        method: a.method,
        headers: a.headers
      }

      try {
        const rawData = (await axios.get(a.url, options)).data
        console.debug('fetchDataWithAdapter', rawData)
        // FIXME Built reducers just once and use. Currently, can't
        // be passed to queue, therefore has to be recreated before
        // every fetch.
        const reducers = buildReducer(a.reducers)
        const data = pipe(...reducers)(rawData)
        checkDataFormat(data)
        return data
      } catch (e) {
        console.error(e)
      }
    })
  )
  console.debug('predefinedFeedJob:allResults', allResults)
  // FIXME: Improve or use flags to throw error when allResults has any undefined variable
  const isValid = allResults.every((r) => r)
  if (!isValid) {
    throw new IcnError(IcnErrorCode.InvalidPriceFeed)
  }
  const aggregatedResults = localAggregatorFn(...allResults)
  console.debug('fetchDataWithAdapter:aggregatedResults', aggregatedResults)

  return aggregatedResults
}

function checkDataFormat(data) {
  if (!data) {
    // check if priceFeed is null, undefined, NaN, "", 0, false
    throw new IcnError(IcnErrorCode.InvalidPriceFeed)
  } else if (!Number.isInteger(data)) {
    // check if priceFeed is not Integer
    throw new IcnError(IcnErrorCode.InvalidPriceFeedFormat)
  }
}

function buildReducer(reducers) {
  return reducers.map((r) => {
    const reducer = reducerMapping[r.function]
    if (!reducer) {
      throw new IcnError(IcnErrorCode.InvalidReducer)
    }
    return reducer(r.args)
  })
}

function extractFeeds(adapter) {
  const adapterId = adapter.id
  const feeds = adapter.feeds.map((f) => {
    return {
      url: f.url,
      headers: f.headers,
      method: f.method,
      reducers: f.reducers
    }
  })

  return { [adapterId]: feeds }
}

function extractAggregators(aggregator) {
  const aggregatorAddress = aggregator.address
  return {
    [aggregatorAddress]: {
      id: aggregator.id,
      address: aggregator.address,
      name: aggregator.name,
      active: aggregator.active,
      fixedHeartbeatRate: aggregator.fixedHeartbeatRate,
      randomHeartbeatRate: aggregator.randomHeartbeatRate,
      threshold: aggregator.threshold,
      absoluteThreshold: aggregator.absoluteThreshold,
      adapterId: aggregator.adapterId
    }
  }
}

function validateAdapter(adapter): IAdapter {
  // TODO extract properties from Interface
  const requiredProperties = ['id', 'active', 'name', 'jobType', 'decimals', 'feeds']
  // TODO show where is the error
  const hasProperty = requiredProperties.map((p) =>
    Object.prototype.hasOwnProperty.call(adapter, p)
  )
  const isValid = hasProperty.every((x) => x)

  if (isValid) {
    return adapter as IAdapter
  } else {
    throw new IcnError(IcnErrorCode.InvalidAdapter)
  }
}

function validateAggregator(adapter): IAggregator {
  // TODO extract properties from Interface
  const requiredProperties = [
    'id',
    'address',
    'active',
    'name',
    'fixedHeartbeatRate',
    'randomHeartbeatRate',
    'threshold',
    'absoluteThreshold',
    'adapterId'
  ]
  // TODO show where is the error
  const hasProperty = requiredProperties.map((p) =>
    Object.prototype.hasOwnProperty.call(adapter, p)
  )
  const isValid = hasProperty.every((x) => x)

  if (isValid) {
    return adapter as IAggregator
  } else {
    throw new IcnError(IcnErrorCode.InvalidAggregator)
  }
}

export function uniform(a: number, b: number): number {
  if (a > b) {
    throw new IcnError(IcnErrorCode.UniformWrongParams)
  }
  return a + Math.round(Math.random() * (b - a))
}
