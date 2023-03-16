import axios from 'axios'
import { Logger } from 'pino'
import { IListenerRawConfig } from '../types'
import { ORAKL_NETWORK_API_URL } from '../settings'
import { buildUrl } from '../utils'
import { OraklError, OraklErrorCode } from '../errors'

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
    const listenerEndpoint = buildUrl(ORAKL_NETWORK_API_URL, 'listener')
    return (await axios.get(listenerEndpoint, { data: { service, chain } }))?.data
  } catch (e) {
    logger?.error({ name: 'getListeners', file: FILE_NAME, ...e }, 'error')
    throw new OraklError(OraklErrorCode.GetListenerRequestFailed)
  }
}
