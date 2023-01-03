import * as Fs from 'node:fs/promises'
import * as Path from 'node:path'
import { reducerMapping } from './reducer'
import { IcnError, IcnErrorCode } from '../errors'
import { loadJson } from '../utils'
import { IAdapter, IAggregator } from '../types'
import { ADAPTER_ROOT_DIR, AGGREGATOR_ROOT_DIR } from '../settings'

export async function loadAdapters() {
  const adapterPaths = await Fs.readdir(ADAPTER_ROOT_DIR)

  const allRawAdapters = await Promise.all(
    adapterPaths.map(async (ap) => validateAdapter(await loadJson(Path.join(ADAPTER_ROOT_DIR, ap))))
  )
  const activeRawAdapters = allRawAdapters.filter((a) => a.active)
  return activeRawAdapters.map((a) => extractFeeds(a))
}

export async function loadAggregators() {
  const aggregatorPaths = await Fs.readdir(AGGREGATOR_ROOT_DIR)

  const allRawAggregators = await Promise.all(
    aggregatorPaths.map(async (ap) =>
      validateAggregator(await loadJson(Path.join(AGGREGATOR_ROOT_DIR, ap)))
    )
  )
  const activeRawAggregators = allRawAggregators.filter((a) => a.active)
  return activeRawAggregators
}

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

function validateAdapter(adapter): IAdapter {
  // TODO extract properties from Interface
  const requiredProperties = ['active', 'name', 'job_type', 'adapter_id', 'feeds']
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
