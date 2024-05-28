import axios from 'axios'
import { Logger } from 'pino'
import { OraklError, OraklErrorCode } from '../errors'
import { ORAKL_NETWORK_API_URL } from '../settings'
import { IListenerRawConfig } from '../types'
import { buildUrl } from '../utils'
import { IBlock } from './types'

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
 * @param {string} service
 * @return {Promise<IBlock>}
 * @exception {FailedToGetObservedBlock}
 */
export async function getObservedBlock({ service }: { service: string }): Promise<IBlock | null> {
  try {
    const endpoint = buildUrl(ORAKL_NETWORK_API_URL, `blocks/observed?service=${service}`)
    return (await axios.get(endpoint))?.data
  } catch (e) {
    throw new OraklError(OraklErrorCode.FailedToGetObservedBlock)
  }
}

/**
 * @param {string} service
 * @return {Promise<IBlocks[]>}
 * @exception {FailedToGetUnprocessedBlocks}
 */
export async function getUnprocessedBlocks({ service }: { service: string }): Promise<IBlock[]> {
  try {
    const endpoint = buildUrl(ORAKL_NETWORK_API_URL, `blocks/unprocessed?service=${service}`)
    return (await axios.get(endpoint))?.data
  } catch (e) {
    throw new OraklError(OraklErrorCode.FailedToGetUnprocessedBlocks)
  }
}

/**
 * @param {string} service
 * @param {number} blockNumber
 * @return {Promise<void>}
 * @exception {FailedInsertUnprocessedBlock}
 */
export async function insertUnprocessedBlocks({
  blocks,
  service
}: {
  service: string
  blocks: number[]
}): Promise<void> {
  try {
    const endpoint = buildUrl(ORAKL_NETWORK_API_URL, `blocks/unprocessed`)
    await axios.post(endpoint, { service, blocks })
  } catch (e) {
    throw new OraklError(OraklErrorCode.FailedInsertUnprocessedBlock)
  }
}

/**
 * @param {string} service
 * @param {number} blockNumber
 * @return {Promise<void>}
 * @exception {FailedDeleteUnprocessedBlock}
 */
export async function deleteUnprocessedBlock({
  blockNumber,
  service
}: {
  blockNumber: number
  service: string
}): Promise<void> {
  try {
    const endpoint = buildUrl(ORAKL_NETWORK_API_URL, `blocks/unprocessed/${service}/${blockNumber}`)
    await axios.delete(endpoint)
  } catch (e) {
    throw new OraklError(OraklErrorCode.FailedDeleteUnprocessedBlock)
  }
}

/**
 * @param {string} service
 * @param {number} blockNumber
 * @return {Promise<IBlock>}
 * @exception {FailedUpsertObservedBlock}
 */
export async function upsertObservedBlock({
  blockNumber,
  service
}: {
  service: string
  blockNumber: number
}): Promise<IBlock> {
  try {
    const endpoint = buildUrl(ORAKL_NETWORK_API_URL, `blocks/observed`)
    return (await axios.post(endpoint, { service, blockNumber }))?.data
  } catch (e) {
    throw new OraklError(OraklErrorCode.FailedUpsertObservedBlock)
  }
}
