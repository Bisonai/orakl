import axios from 'axios'
import { Logger } from 'pino'
import { OraklError, OraklErrorCode } from './errors'
import {
  CHAIN,
  DATA_FEED_SERVICE_NAME,
  L2_CHAIN,
  L2_DATA_FEED_SERVICE_NAME,
  ORAKL_NETWORK_API_URL,
} from './settings'
import { IReporterConfig, IVrfConfig } from './types'
import { buildUrl } from './utils'

const FILE_NAME = import.meta.url

/**
 * Fetch all VRF keys from Orakl Network API given a `chain` name.
 *
 * @param {string} chain name
 * @param {pino.Logger} logger
 * @return {Promise<IListenerRawConfig[]>} raw listener configuration
 * @exception {GetVrfConfigRequestFailed}
 */
export async function getVrfConfig({
  chain,
  logger,
}: {
  chain: string
  logger?: Logger
}): Promise<IVrfConfig> {
  try {
    const endpoint = buildUrl(ORAKL_NETWORK_API_URL, 'vrf')
    const vrfKeys = (await axios.get(endpoint, { data: { chain } }))?.data

    if (vrfKeys.length == 0) {
      throw new Error(`Found no VRF key for chain [${chain}]`)
    } else if (vrfKeys.length > 1) {
      throw new Error(`Found more than one VRF key for chain [${chain}]`)
    }

    return vrfKeys[0]
  } catch (e) {
    logger?.error({ name: 'getVrfConfig', file: FILE_NAME, ...e }, 'error')
    throw new OraklError(OraklErrorCode.GetVrfConfigRequestFailed)
  }
}

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
  logger,
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
    if (e.code === 'ECONNREFUSED') {
      throw new OraklError(OraklErrorCode.FailedToConnectAPI)
    } else {
      throw new OraklError(OraklErrorCode.GetReporterRequestFailed)
    }
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
  logger,
}: {
  id: string
  logger?: Logger
}): Promise<IReporterConfig> {
  try {
    const endpoint = buildUrl(ORAKL_NETWORK_API_URL, `reporter/${id}`)
    return (await axios.get(endpoint))?.data
  } catch (e) {
    logger?.error({ name: 'getReporters', file: FILE_NAME, ...e }, 'error')
    if (e.code === 'ECONNREFUSED') {
      throw new OraklError(OraklErrorCode.FailedToConnectAPI)
    } else {
      throw new OraklError(OraklErrorCode.GetReporterRequestFailed)
    }
  }
}

/**
 * Fetch reporter from the Orakl Network API that are associated with
 * given `service` and `chain`.
 *
 * @param {string} service name
 * @param {string} chain name
 * @param {string} oracle address
 * @param {pino.Logger} logger
 * @return {IReporterConfig} reporter configuration
 * @exception {GetReporterRequestFailed}
 */
export async function getReporterByOracleAddress({
  service,
  chain,
  oracleAddress,
  logger,
}: {
  service: string
  chain: string
  oracleAddress: string
  logger: Logger
}): Promise<IReporterConfig> {
  try {
    const endpoint = buildUrl(ORAKL_NETWORK_API_URL, `reporter/oracle-address/${oracleAddress}`)
    const reporter = (await axios.get(endpoint, { data: { service, chain } }))?.data

    if (reporter.length != 1) {
      logger.error(`Expected 1 reporter, received ${reporter.length}`)
      throw new Error()
    }

    return reporter[0]
  } catch (e) {
    logger.error({ name: 'getReportersByOracleAddress', file: FILE_NAME, ...e }, 'error')
    if (e.code === 'ECONNREFUSED') {
      throw new OraklError(OraklErrorCode.FailedToConnectAPI)
    } else {
      throw new OraklError(OraklErrorCode.GetReporterRequestFailed)
    }
  }
}

/**
 * Get address of node operator given an `oracleAddress`. The data are
 * fetched from the Orakl Network API.
 *
 * @param {string} oracle address
 * @return {string} address of node operator
 * @exception {OraklErrorCode.GetReporterRequestFailed} raises when request failed
 */
export async function getOperatorAddress({
  oracleAddress,
  logger,
}: {
  oracleAddress: string
  logger: Logger
}) {
  logger.debug('getOperatorAddress')

  return await (
    await getReporterByOracleAddress({
      service: DATA_FEED_SERVICE_NAME,
      chain: CHAIN,
      oracleAddress,
      logger,
    })
  ).address
}

export async function getOperatorAddressL2({
  oracleAddress,
  logger,
}: {
  oracleAddress: string
  logger: Logger
}) {
  logger.debug('getOperatorAddressL2')

  return await (
    await getReporterByOracleAddress({
      service: L2_DATA_FEED_SERVICE_NAME,
      chain: L2_CHAIN,
      oracleAddress,
      logger,
    })
  ).address
}
