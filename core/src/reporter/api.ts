import axios from 'axios'
import { Logger } from 'pino'
import { IReporterConfig } from '../types'
import { ORAKL_NETWORK_API_URL } from '../settings'
import { buildUrl } from '../utils'
import { OraklError, OraklErrorCode } from '../errors'

const FILE_NAME = import.meta.url

/**
 * Fetch reporters from the Orakl Network API that are associated with
 * given `service` and `chain`.
 *
 * @param {string} service name
 * @param {string} chain name
 * @param {pino.Logger} logger
 * @return {IReporterConfig} reporter configuration
 * @exception {GetReporterRequestFailed}
 */
export async function getReporters({
  service,
  chain,
  logger
}: {
  service?: string
  chain?: string
  logger?: Logger
}): Promise<IReporterConfig[]> {
  try {
    const endpoint = buildUrl(ORAKL_NETWORK_API_URL, 'reporter')
    return (await axios.get(endpoint, { data: { service, chain } }))?.data
  } catch (e) {
    logger?.error({ name: 'getReporters', file: FILE_NAME, ...e }, 'error')
    throw new OraklError(OraklErrorCode.GetReporterRequestFailed)
  }
}

/**
 * Fetch single reporter given its ID from the Orakl Network API.
 *
 * @param {string} reporter ID
 * @param {pino.Logger} logger
 * @return {IReporterConfig} reporter configuration
 * @exception {GetReporterRequestFailed}
 */
export async function getReporter({
  id,
  logger
}: {
  id: string
  logger?: Logger
}): Promise<IReporterConfig> {
  try {
    const endpoint = buildUrl(ORAKL_NETWORK_API_URL, `reporter/${id}`)
    return (await axios.get(endpoint))?.data
  } catch (e) {
    logger?.error({ name: 'getReporter', file: FILE_NAME, ...e }, 'error')
    throw new OraklError(OraklErrorCode.GetReporterRequestFailed)
  }
}
