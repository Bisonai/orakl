import axios from 'axios'
import { URL } from 'node:url'
import { Logger } from 'pino'
import { OraklError, OraklErrorCode } from '../errors'
import { ORAKL_NETWORK_API_URL } from '../settings'
import { IAggregator, IErrorMsgData, IL2AggregatorPair } from '../types'
import { buildUrl } from '../utils'

export const AGGREGATE_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'aggregate')
export const AGGREGATOR_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'aggregator')
export const ERROR_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'error')
export const L2_AGGREGATOR_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'l2aggregator')

/**
 * Get all `Aggregator`s on given `chain`. The data are fetched from
 * the `Orakl Network API`.
 *
 * @param {string} chain name
 * @param {string} activeness of aggregator
 * @param {Logger} logger
 * @return {Aggregator[]}
 * @exception {FailedToGetAggregator}
 */
export async function getAggregators({
  chain,
  active,
  logger,
}: {
  chain: string
  active?: boolean
  logger: Logger
}): Promise<IAggregator[]> {
  try {
    const url = new URL(AGGREGATOR_ENDPOINT)
    url.searchParams.append('chain', chain)
    if (active) {
      url.searchParams.append('active', 'true')
    }
    const response = (await axios.get(url.toString()))?.data
    return response
  } catch (e) {
    logger.error(e)
    throw new OraklError(OraklErrorCode.FailedToGetAggregator)
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
  logger,
}: {
  aggregatorHash: string
  chain: string
  logger: Logger
}): Promise<IAggregator> {
  try {
    const url = buildUrl(AGGREGATOR_ENDPOINT, `${aggregatorHash}/${chain}`)
    const response = (await axios.get(url))?.data
    return response
  } catch (e) {
    logger.error(e)
    throw new OraklError(OraklErrorCode.FailedToGetAggregator)
  }
}

/**
 * Store catched RR worker error log to
 * `Orakl Network API` `error` endpoint
 *
 * @param {data} IErrorMsgData
 * @param {Logger} logger
 * @exception {FailedToGetAggregate}
 */
export async function storeErrorMsg({ data, logger }: { data: IErrorMsgData; logger: Logger }) {
  try {
    const response = (await axios.post(ERROR_ENDPOINT, data))?.data
    return response
  } catch (e) {
    logger.error(e)
    throw new OraklError(OraklErrorCode.FailedToStoreErrorMsg)
  }
}

/**
 * Get L2 oracle address associated with L1 oracle address
 * @param {string} L1 oracle address
 * @returns {string} L2 oracle address
 */
export async function getL2AddressGivenL1Address({
  oracleAddress,
  chain,
  logger,
}: {
  oracleAddress: string
  chain: string
  logger: Logger
}): Promise<IL2AggregatorPair> {
  try {
    const url = buildUrl(L2_AGGREGATOR_ENDPOINT, `${chain}/${oracleAddress}`)
    const response = (await axios.get(url))?.data
    return response
  } catch (e) {
    logger.error(e)
    throw new OraklError(OraklErrorCode.FailedToGetAggregator)
  }
}
