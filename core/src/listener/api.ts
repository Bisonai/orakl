import axios from 'axios'
import { Logger } from 'pino'
import { OraklError, OraklErrorCode } from '../errors'
import { ORAKL_NETWORK_API_URL } from '../settings'
import { IListenerRawConfig } from '../types'
import { buildUrl } from '../utils'
import { IObservedBlock } from './types'

const FILE_NAME = import.meta.url

/**
 * Fetch listeners from the Orakl Network API that are associated with
 * given `service` and `chain`.
 *
 * @param {string} service name
 * @param {string} chain name
 * @param {pino.Logger} logger
 * @return {Promise<IListenerRawConfig[]>} raw listener configuration
 * @exception {GetListenerRequestFailed}
 */
export async function getListeners({
  service,
  chain,
  logger
}: {
  service?: string
  chain?: string
  logger?: Logger
}): Promise<IListenerRawConfig[]> {
  try {
    const endpoint = buildUrl(ORAKL_NETWORK_API_URL, 'listener')
    return (await axios.get(endpoint, { data: { service, chain } }))?.data
  } catch (e) {
    logger?.error({ name: 'getListeners', file: FILE_NAME, ...e }, 'error')
    throw new OraklError(OraklErrorCode.GetListenerRequestFailed)
  }
}

/**
 * Fetch single listener given its ID from the Orakl Network API.
 *
 * @param {string} listener ID
 * @param {pino.Logger} logger
 * @return {Promise<IListenerRawConfig>} raw listener configuration
 * @exception {GetListenerRequestFailed}
 */
export async function getListener({
  id,
  logger
}: {
  id: string
  logger?: Logger
}): Promise<IListenerRawConfig> {
  try {
    const endpoint = buildUrl(ORAKL_NETWORK_API_URL, `listener/${id}`)
    return (await axios.get(endpoint))?.data
  } catch (e) {
    logger?.error({ name: 'getListener', file: FILE_NAME, ...e }, 'error')
    throw new OraklError(OraklErrorCode.GetListenerRequestFailed)
  }
}

/**
 * Get observed block number from the Orakl Network API for a given contract address
 *
 * @param {string} blockKey
 * @param {pino.Logger} logger
 * @return {Promise<IObservedBlock>}
 * @exception {OraklErrorCode.GetObservedBlockFailed}
 */
export async function getObservedBlock({
  blockKey,
  logger
}: {
  blockKey: string
  logger?: Logger
}): Promise<IObservedBlock> {
  try {
    const endpoint = buildUrl(ORAKL_NETWORK_API_URL, `listener/observed-block?blockKey=${blockKey}`)
    return (await axios.get(endpoint))?.data
  } catch (e) {
    logger?.error({ name: 'getObservedBlock', file: FILE_NAME, ...e }, 'error')
    throw new OraklError(OraklErrorCode.GetObservedBlockFailed)
  }
}

/**
 * Upsert listener observed block number to the Orakl Network API for a given contract address
 *
 * @param {string} blockKey
 * @param {number} blockNumber
 * @param {pino.Logger} logger
 * @return {Promise<IObservedBlock>}
 * @exception {OraklErrorCode.UpsertObservedBlockFailed}
 */
export async function upsertObservedBlock({
  blockKey,
  blockNumber,
  logger
}: {
  blockKey: string
  blockNumber: number
  logger?: Logger
}): Promise<IObservedBlock> {
  try {
    const endpoint = buildUrl(ORAKL_NETWORK_API_URL, 'listener/observed-block')
    return (await axios.post(endpoint, { blockKey, blockNumber }))?.data
  } catch (e) {
    logger?.error({ name: 'upsertObservedBlock', file: FILE_NAME, ...e }, 'error')
    throw new OraklError(OraklErrorCode.UpsertObservedBlockFailed)
  }
}
