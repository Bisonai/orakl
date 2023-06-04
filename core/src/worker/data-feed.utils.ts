import { ethers } from 'ethers'
import { Logger } from 'pino'
import { IOracleRoundState, IRoundData, IDataFeedTransactionParameters } from '../types'
import { PROVIDER, MAX_DATA_STALENESS } from '../settings'
import { Aggregator__factory } from '@bisonai/orakl-contracts'

/**
 * Compute the number of seconds until the next round.
 *
 * @param {string} oracle address
 * @param {number} heartbeat
 * @param {Logger}
 * @return {number} delay in seconds until the next round
 */
export async function getSynchronizedDelay({
  oracleAddress,
  heartbeat,
  logger
}: {
  oracleAddress: string
  heartbeat: number
  logger: Logger
}): Promise<number> {
  logger.debug('getSynchronizedDelay')

  const { startedAt } = await currentRoundStartedAtCall({
    oracleAddress,
    logger
  })

  const delay = heartbeat - (startedAt % heartbeat)
  logger.debug({ heartbeat, delay, startedAt })

  return delay
}

async function currentRoundStartedAtCall({
  oracleAddress,
  logger
}: {
  oracleAddress: string
  logger?: Logger
}) {
  logger?.debug({ oracleAddress }, 'currentRoundStartedAtCall')
  const aggregator = new ethers.Contract(oracleAddress, Aggregator__factory.abi, PROVIDER)
  const startedAt = await aggregator.currentRoundStartedAt()
  logger?.debug({ startedAt }, 'startedAt')
  return startedAt
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

export function isStale({ timestamp, logger }: { timestamp: string; logger: Logger }) {
  const now = Date.now()
  const fetchedAt = Date.parse(timestamp)
  const dataStaleness = Math.max(0, now - fetchedAt)
  logger.debug(`Data staleness ${dataStaleness} ms`)
  return dataStaleness > MAX_DATA_STALENESS
}

export function buildTransaction({
  payloadParameters,
  to,
  gasMinimum,
  iface,
  logger
}: {
  payloadParameters: IDataFeedTransactionParameters
  to: string
  gasMinimum: number
  iface: ethers.utils.Interface
  logger: Logger
}) {
  const { roundId, submission } = payloadParameters
  const payload = iface.encodeFunctionData('submit', [roundId, submission])
  const gasLimit = gasMinimum
  const tx = {
    payload,
    gasLimit,
    to
  }
  logger.debug(tx)
  return tx
}
