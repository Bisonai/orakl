import * as Fs from 'node:fs/promises'
import * as Path from 'node:path'
import { got } from 'got'
import { reducerMapping } from './reducer'
import { IcnError, IcnErrorCode } from '../errors'
import { pipe, loadJson } from '../utils'
import { IAdapter, IAggregator } from '../types'
import { localAggregatorFn, ADAPTER_ROOT_DIR, AGGREGATOR_ROOT_DIR } from '../settings'

export async function loadAdapters() {
  const adapterPaths = await Fs.readdir(ADAPTER_ROOT_DIR)

  const allRawAdapters = await Promise.all(
    adapterPaths.map(async (ap) => validateAdapter(await loadJson(Path.join(ADAPTER_ROOT_DIR, ap))))
  )
  const activeRawAdapters = allRawAdapters.filter((a) => a.active)
  return Object.assign({}, ...activeRawAdapters.map((a) => extractFeeds(a)))
}

export async function loadAggregators() {
  const aggregatorPaths = await Fs.readdir(AGGREGATOR_ROOT_DIR)

  const allRawAggregators = await Promise.all(
    aggregatorPaths.map(async (ap) =>
      validateAggregator(await loadJson(Path.join(AGGREGATOR_ROOT_DIR, ap)))
    )
  )
  const activeRawAggregators = allRawAggregators.filter((a) => a.active)
  return Object.assign({}, ...activeRawAggregators.map((a) => extractAggregators(a)))
}

export function mergeAggregatorsAdapters(aggregators, adapters) {
  // FIXME use mapping instead
  let aggregatorsWithAdapters: any = [] // TODO replace any

  for (const agId in aggregators) {
    const ag = aggregators[agId]
    if (ag) {
      const ad = adapters[ag.adapterId]

      ag['aggregatorId'] = agId
      ag['adapter'] = ad

      aggregatorsWithAdapters.push(ag)
    } else {
      throw new IcnError(IcnErrorCode.MissingAdapter)
    }
  }

  return aggregatorsWithAdapters
}

export async function fetchDataWithAdapter(adapter) {
  const allResults = await Promise.all(
    adapter.map(async (a) => {
      const options = {
        method: a.method,
        headers: a.headers
      }

      try {
        const rawData = await got(a.url, options).json()
        console.debug('fetchDataWithAdapter', rawData)
        // FIXME Built reducers just once and use. Currently, can't
        // be passed to queue, therefore has to be recreated before
        // every fetch.
        const reducers = buildReducer(a.reducers)
        return pipe(...reducers)(rawData)
      } catch (e) {
        console.error(e)
      }
    })
  )
  console.debug('predefinedFeedJob:allResults', allResults)

  const aggregatedResults = localAggregatorFn(...allResults)
  console.debug('fetchDataWithAdapter:aggregatedResults', aggregatedResults)

  return aggregatedResults
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
  const adapterId = adapter.adapterId
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
  const aggregatorId = aggregator.id
  return {
    [aggregatorId]: {
      name: aggregator.name,
      active: aggregator.active,
      aggregatorAddress: aggregator.aggregatorAddress,
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
  const requiredProperties = ['active', 'name', 'jobType', 'adapterId', 'feeds']
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
    'active',
    'name',
    'aggregatorAddress',
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
