import { Logger } from 'pino'
import { getReporterByOracleAddress } from '../api'
import { CHAIN, DATA_FEED_SERVICE_NAME } from '../settings'

/**
 * Get address of node operator given an `oracleAddress`. The data are fetched from the Orakl Network API.
 *
 * @param {string} oracle address
 * @return {string} address of node operator
 * @exception {OraklErrorCode.GetReporterRequestFailed} raises when request failed
 */
export async function getOperatorAddress({
  oracleAddress,
  logger
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
      logger
    })
  ).address
}
