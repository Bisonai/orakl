import { ethers } from 'ethers'
import { Logger } from 'pino'
import { IOracleRoundState, IRoundData } from '../types'
import { getReporterByOracleAddress } from '../api'
import { PROVIDER } from '../settings'
import { CHAIN, DATA_FEED_SERVICE_NAME } from '../settings'
import { Aggregator__factory } from '@bisonai/orakl-contracts'

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

/**
 * Compute the number of seconds until the next round.
 *
 * FIXME modify aggregator to use single contract call
 *
 * @param {string} aggregator address
 * @param {number} heartbeat
 * @param {Logger}
 * @return {number} delay in seconds until the next round
 */
export async function getSynchronizedDelay({
  oracleAddress,
  operatorAddress,
  heartbeat,
  logger
}: {
  oracleAddress: string
  operatorAddress: string
  heartbeat: number
  logger: Logger
}): Promise<number> {
  logger.debug('getSynchronizedDelay')

  let startedAt = 0
  const { _startedAt, _roundId } = await oracleRoundStateCall({
    oracleAddress,
    operatorAddress,
    logger
  })

  if (_startedAt.toNumber() != 0) {
    startedAt = _startedAt.toNumber()
  } else {
    const { _startedAt } = await oracleRoundStateCall({
      oracleAddress,
      operatorAddress,
      roundId: Math.max(0, _roundId - 1)
    })
    startedAt = _startedAt.toNumber()
  }

  logger.debug({ startedAt }, 'startedAt')
  const delay = heartbeat - (startedAt % heartbeat)
  logger.debug({ delay }, 'delay')

  return delay
}

export async function oracleRoundStateCall({
  oracleAddress,
  operatorAddress,
  logger,
  roundId
}: {
  oracleAddress: string
  operatorAddress: string
  roundId?: number
  logger?: Logger
}): Promise<IOracleRoundState> {
  logger?.debug({ oracleAddress, operatorAddress }, 'oracleRoundStateCall')

  const aggregator = new ethers.Contract(oracleAddress, Aggregator__factory.abi, PROVIDER)

  let queriedRoundId = 0
  if (roundId) {
    queriedRoundId = roundId
  }

  return await aggregator.oracleRoundState(operatorAddress, queriedRoundId)
}

export async function getRoundDataCall({
  oracleAddress,
  roundId
}: {
  oracleAddress: string
  roundId: number
}): Promise<IRoundData> {
  const aggregator = new ethers.Contract(oracleAddress, Aggregator__factory.abi, PROVIDER)
  return await aggregator.getRoundData(roundId)
}
