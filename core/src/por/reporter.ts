import { Aggregator__factory } from '@bisonai/orakl-contracts'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import { getReporterByOracleAddress } from '../api'
import { buildWallet, sendTransaction } from '../reporter/utils'
import {
  CHAIN,
  DATA_FEED_FULFILL_GAS_MINIMUM,
  DATA_FEED_SERVICE_NAME,
  PROVIDER,
  PROVIDER_URL
} from '../settings'
import { IReporterConfig } from '../types'
import { buildTransaction } from '../worker/data-feed.utils'

async function checkSubmissionFrequency() {
  // TODO
  return true
}

async function checkDeviationThreashold() {
  // TODO
  return true
}

async function report({
  value,
  oracleAddress,
  logger
}: {
  value: bigint
  oracleAddress: string
  logger: Logger
}) {
  const reporter: IReporterConfig = await getReporterByOracleAddress({
    service: DATA_FEED_SERVICE_NAME,
    chain: CHAIN,
    oracleAddress,
    logger: logger
  })

  console.log('Reporter:', reporter)
  const iface = new ethers.utils.Interface(Aggregator__factory.abi)
  const aggregator = new ethers.Contract(oracleAddress, Aggregator__factory.abi, PROVIDER)
  let queriedRoundId = 0
  const state = await aggregator.oracleRoundState(reporter.address, queriedRoundId)
  const roundId = state._roundId
  const getLastestRoundData = await aggregator.getRoundData(roundId - 1)

  const tx = buildTransaction({
    payloadParameters: {
      roundId,
      submission: value
    },
    to: oracleAddress,
    gasMinimum: DATA_FEED_FULFILL_GAS_MINIMUM,
    iface,
    logger
  })

  const wallet = await buildWallet({ privateKey: reporter.privateKey, providerUrl: PROVIDER_URL })
  const txParams = { wallet, ...tx, logger }
  await sendTransaction(txParams)
}

export async function reportData({
  value,
  oracleAddress,
  logger
}: {
  value: bigint
  oracleAddress: string
  logger: Logger
}) {
  if ((await checkSubmissionFrequency()) || (await checkDeviationThreashold())) {
    await report({ value, oracleAddress, logger })
  }
}
