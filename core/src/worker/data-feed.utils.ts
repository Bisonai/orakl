import { ethers } from 'ethers'
import { Logger } from 'pino'
import { IOracleRoundState, IRoundData } from '../types'
import { getReporterByOracleAddress } from '../api'
import { PROVIDER } from '../settings'
import { CHAIN, DATA_FEED_SERVICE_NAME } from '../settings'
import { Aggregator__factory } from '@bisonai/orakl-contracts'

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

  const { startedAt, roundId } = await oracleRoundStateCall({
    oracleAddress,
    operatorAddress,
    logger
  })

  let startTime = startedAt.toNumber()

  if (startTime == 0) {
    const { startedAt } = await oracleRoundStateCall({
      oracleAddress,
      operatorAddress,
      roundId: Math.max(0, roundId - 1)
    })
    startTime = startedAt.toNumber()
  }

  const delay = heartbeat - (startTime % heartbeat)
  logger.debug({ heartbeat, delay, startTime })

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

  const state = await aggregator.oracleRoundState(operatorAddress, queriedRoundId)
  return {
    eligibleToSubmit: state._eligibleToSubmit,
    roundId: state._roundId,
    latestSubmission: state._latestSubmission,
    startedAt: state._startedAt,
    timeout: state._timeout,
    availableFunds: state._availableFunds,
    oracleCount: state._oracleCount,
    paymentAmount: state._paymentAmount
  }
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
