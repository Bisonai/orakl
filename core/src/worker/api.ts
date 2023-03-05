import axios from 'axios'
import { Logger } from 'pino'
import { IAggregatorMetadata } from '../types'
import { IcnError, IcnErrorCode } from '../errors'
import { ORAKL_API_DATA_FEED_ENDPOINT } from '../settings'

/* Fetch data from Orakl API Data Feed endpoint given aggregator ID.
 *
 * @param {string} aggregator ID
 * @param {Logger} logger
 * @return {number} the latest aggregated value
 * @exception {FailedToFetchFromDataFeed} raised when Orakl Network
 * API does not respond or responds in an unexpected format.
 */
export async function fetchDataFeed({
  id,
  logger
}: {
  id: string
  logger: Logger
}): Promise<number> {
  try {
    const url = [ORAKL_API_DATA_FEED_ENDPOINT, id].join('/')
    logger.debug({ url }, 'data-feed-url')
    const value = (await axios.get(url)).data
    logger.debug({ value }, 'data-feed-value')
    // TODO check for value type
    return value
  } catch (e) {
    logger.error(e)
    throw new IcnError(IcnErrorCode.FailedToFetchFromDataFeed)
  }
}

/* Fetch metadata about given aggregator ID from Orakl API.
 *
 * @param {string} aggregator ID
 * @param {Logger} logger
 * @return {Promise<IDataFeedMetadata>} data feed metadata
 * @exception {FailedToFetchFromDataFeed} raised when Orakl Network
 * API does not respond or responds in an unexpected format.
 */
export async function fetchAggregatorMetadata({
  id,
  logger
}: {
  id: string
  logger: Logger
}): Promise<IAggregatorMetadata> {
  try {
    const url = [ORAKL_API_DATA_FEED_ENDPOINT, id, 'metadata'].join('/')
    logger.debug({ url }, 'data-feed-url')
    const metadata = (await axios.get(url)).data
    logger.debug({ metadata }, 'data-feed-metadata')
    // TODO check for IDataFeedMetadata type
    return metadata as IAggregatorMetadata
  } catch (e) {
    logger.error(e)
    throw new IcnError(IcnErrorCode.FailedToFetchFromDataFeed)
  }
}
