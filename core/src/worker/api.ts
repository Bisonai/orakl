import axios from 'axios'
import { URL } from 'node:url'
import { Logger } from 'pino'
import { IAggregatorNew, IAggregate } from '../types'
import { IcnError, IcnErrorCode } from '../errors'
import { ORAKL_NETWORK_API_URL } from '../settings'
import { buildUrl } from '../utils'

export const AGGREGATE_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'api/v1/aggregate')
export const AGGREGATOR_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'api/v1/aggregator')

/**
 * Fetch aggregate data from `Orakl Network API` data feed endpoint
 * given aggregator ID.
 *
 * @param {string} aggregator hash
 * @param {Logger} logger
 * @return {number} the latest aggregated value
 * @exception {FailedToGetAggregate}
 */
export async function fetchDataFeed({
  aggregatorHash,
  logger
}: {
  aggregatorHash: string
  logger: Logger
}): Promise<IAggregate> {
  try {
    const url = buildUrl(AGGREGATE_ENDPOINT, `${aggregatorHash}/latest`)
    const response = (await axios.get(url))?.data
    return response
  } catch (e) {
    logger.error(e)
    throw new IcnError(IcnErrorCode.FailedToGetAggregate)
  }
}

/**
 * Get `Aggregator` given aggregator address.
 *
 * @param {string} aggregator address
 * @param {Logger} logger
 * @return {Aggregator}
 * @exception {FailedToGetAggregate}
 */
export async function getAggregatorGivenAddress({
  aggregatorAddress,
  logger
}: {
  aggregatorAddress: string
  logger: Logger
}): Promise<IAggregatorNew> {
  try {
    const url = new URL(AGGREGATOR_ENDPOINT)
    url.searchParams.append('address', aggregatorAddress)
    const response = (await axios.get(url.toString()))?.data
    if (response.length == 0) {
      const msg = 'nothing found'
      console.log(msg)
      throw new Error(msg)
    } else if (response.length == 1) {
      logger.debug(response)
      return response[0]
    } else {
      console.dir(response, { depth: null })
      const msg = 'too many found'
      console.log(msg)
      throw new Error(msg)
    }
  } catch (e) {
    logger.error(e)
    throw new IcnError(IcnErrorCode.FailedToGetAggregator)
  }
}

/**
 * Get all active `Aggregator`s from `Orakl API` given an aggregator
 * hash and chain.
 *
 * @param {string} chain name
 * @param {Logger} logger
 * @return {Aggregator}
 * @exception {FailedToGetAggregate}
 */
export async function getActiveAggregators({
  chain,
  logger
}: {
  chain: string
  logger: Logger
}): Promise<IAggregatorNew[]> {
  try {
    const url = new URL(AGGREGATOR_ENDPOINT)
    url.searchParams.append('active', 'true')
    url.searchParams.append('chain', chain)
    const response = (await axios.get(url.toString()))?.data
    return response
  } catch (e) {
    logger.error(e)
    throw new IcnError(IcnErrorCode.FailedToGetAggregator)
  }
}

/**
 * Get `Aggregator` from `Orakl API` given an aggregator hash and chain.
 *
 * @param {string} aggregator hash
 * @param {string} chain name
 * @param {Logger} logger
 * @return {Aggregator}
 * @exception {FailedToGetAggregator}
 */
export async function getAggregator({
  aggregatorHash,
  chain,
  logger
}: {
  aggregatorHash: string
  chain: string
  logger: Logger
}): Promise<IAggregatorNew> {
  try {
    const url = buildUrl(AGGREGATOR_ENDPOINT, `${aggregatorHash}/${chain}`)
    const response = (await axios.get(url))?.data
    return response
  } catch (e) {
    logger.error(e)
    throw new IcnError(IcnErrorCode.FailedToGetAggregator)
  }
}
