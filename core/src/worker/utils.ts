import { ethers } from 'ethers'
import { Logger } from 'pino'
import { OraklError, OraklErrorCode } from '../errors'
import { IOracleRoundState, IRoundData } from '../types'
import { PROVIDER } from '../settings'
import { Aggregator__factory } from '@bisonai/orakl-contracts'

export function buildReducer(reducerMapping, reducers) {
  return reducers.map((r) => {
    const reducer = reducerMapping[r.function]
    if (!reducer) {
      throw new OraklError(OraklErrorCode.InvalidReducer)
    }
    return reducer(r?.args)
  })
}

export function uniform(a: number, b: number): number {
  if (a > b) {
    throw new OraklError(OraklErrorCode.UniformWrongParams)
  }
  return a + Math.round(Math.random() * (b - a))
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
