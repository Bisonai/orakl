import axios from 'axios'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import { dataFeedReducerMapping } from './reducer'
import { IcnError, IcnErrorCode } from '../errors'
import { pipe } from '../utils'
import { IAdapter, IAggregator, IOracleRoundState, IRoundData } from '../types'
import { getAdapters, getAggregators, localAggregatorFn, DB, CHAIN, PROVIDER } from '../settings'
import { Aggregator__factory } from '@bisonai/orakl-contracts'

const FILE_NAME = import.meta.url

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
    const aggregator = aggregators[agAddress]
    if (!aggregator) {
      throw new IcnError(IcnErrorCode.MissingAggregator)
    }

    const adapter = adapters[aggregator?.adapterId]
    if (!adapter) {
      throw new IcnError(IcnErrorCode.MissingAdapter)
    }

    aggregator.decimals = adapter.decimals
    aggregator.adapter = adapter.feegregatords

    aggregatorsWithAdapters.push({ [agAddress]: aggregator })
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
export async function fetchDataWithAdapter(adapter, logger?: Logger) {
  const allResults = await Promise.all(
    adapter.map(async (a) => {
      const options = {
        method: a.method,
        headers: a.headers
      }

      try {
        const rawData = (await axios.get(a.url, options)).data
        logger?.debug('fetchDataWithAdapter', rawData)
        // FIXME Built reducers just once and use. Currently, can't
        // be passed to queue, therefore has to be recreated before
        // every fetch.
        const reducers = buildReducer(dataFeedReducerMapping, a.reducers)
        const data = pipe(...reducers)(rawData)
        checkDataFormat(data)
        return data
      } catch (e) {
        logger?.error(e)
      }
    })
  )
  logger?.debug({ name: 'predefinedFeedJob', ...allResults }, 'allResults')
  // FIXME: Improve or use flags to throw error when allResults has any undefined variable
  const isValid = allResults.every((r) => r)
  if (!isValid) {
    throw new IcnError(IcnErrorCode.InvalidPriceFeed)
  }
  const aggregatedResults = localAggregatorFn(...allResults)
  logger?.debug({ name: 'fetchDataWithAdapter', ...aggregatedResults }, 'aggregatedResults')

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

export function buildReducer(reducerMapping, reducers) {
  return reducers.map((r) => {
    const reducer = reducerMapping[r.function]
    if (!reducer) {
      throw new IcnError(IcnErrorCode.InvalidReducer)
    }
    return reducer(r?.args)
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

  return { [adapterId]: { decimals: adapter.decimals, feeds } }
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

export async function oracleRoundStateCall({
  aggregatorAddress,
  operatorAddress,
  logger,
  roundId
}: {
  aggregatorAddress: string
  operatorAddress: string
  roundId?: number
  logger?: Logger
}): Promise<IOracleRoundState> {
  logger?.debug({ name: 'oracleRoundStateCall', file: FILE_NAME })

  const aggregator = new ethers.Contract(aggregatorAddress, Aggregator__factory.abi, PROVIDER)

  let queriedRoundId = 0
  if (roundId) {
    queriedRoundId = roundId
  }

  return await aggregator.oracleRoundState(operatorAddress, queriedRoundId)
}

export async function getRoundDataCall({
  aggregatorAddress,
  roundId
}: {
  aggregatorAddress: string
  roundId: number
}): Promise<IRoundData> {
  const aggregator = new ethers.Contract(aggregatorAddress, Aggregator__factory.abi, PROVIDER)
  return await aggregator.getRoundData(roundId)
}
