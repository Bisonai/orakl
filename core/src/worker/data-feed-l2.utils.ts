import { IOracleRoundState } from '../types'
import { ethers } from 'ethers'
import { L2_PROVIDER_URL } from '../settings'
import { Aggregator__factory } from '@bisonai/orakl-contracts'
import { Logger } from 'pino'

export async function oracleRoundStateCallL2({
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
  const provider = new ethers.providers.JsonRpcProvider(L2_PROVIDER_URL)
  const aggregator = new ethers.Contract(oracleAddress, Aggregator__factory.abi, provider)

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
