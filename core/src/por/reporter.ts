import { Aggregator__factory } from '@bisonai/orakl-contracts'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import { getReporterByOracleAddress } from '../api'
import { buildWallet, sendTransaction } from '../reporter/utils'
import {
  CHAIN,
  DATA_FEED_FULFILL_GAS_MINIMUM,
  POR_SERVICE_NAME,
  PROVIDER,
  PROVIDER_URL
} from '../settings'
import { IAggregator, IReporterConfig } from '../types'
import { buildTransaction } from '../worker/data-feed.utils'

async function shouldReport({
  aggregator,
  value,
  logger
}: {
  aggregator: IAggregator
  value: bigint
  logger: Logger
}) {
  const contract = new ethers.Contract(aggregator.address, Aggregator__factory.abi, PROVIDER)
  const latestRoundData = await contract.latestRoundData()

  // Check Submission Hearbeat
  const updatedAt = Number(latestRoundData.updatedAt) * 1000 // convert to milliseconds
  const now = Date.now()
  const heartbeat = aggregator.heartbeat

  if (updatedAt + heartbeat < now) {
    logger.info('Should report by heartbeat check')
    logger.info(`Last submission time:${updatedAt}, heartbeat:${heartbeat}`)
    return true
  }

  // Check deviation threashold
  if (aggregator.threshold && latestRoundData.answer) {
    const latestSubmission = Number(latestRoundData.answer)
    const currentSubmission = Number(value)

    const range = latestSubmission * aggregator.threshold
    const l = latestSubmission - range
    const r = latestSubmission + range

    if (currentSubmission < l || currentSubmission > r) {
      logger.info('Should report by deviation check')
      logger.info(`Latest submission:${latestSubmission}, currentSubmission:${currentSubmission}`)
      return true
    }
  }
  return false
}

async function submit({
  value,
  oracleAddress,
  logger
}: {
  value: bigint
  oracleAddress: string
  logger: Logger
}) {
  const reporter: IReporterConfig = await getReporterByOracleAddress({
    service: POR_SERVICE_NAME,
    chain: CHAIN,
    oracleAddress,
    logger: logger
  })

  const iface = new ethers.utils.Interface(Aggregator__factory.abi)
  const contract = new ethers.Contract(oracleAddress, Aggregator__factory.abi, PROVIDER)
  const queriedRoundId = 0
  const state = await contract.oracleRoundState(reporter.address, queriedRoundId)
  const roundId = state._roundId

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

  const NUM_TRANSACTION_TRIALS = 3
  for (let i = 0; i < NUM_TRANSACTION_TRIALS; ++i) {
    logger.info(`Reporting to round:${roundId} with value:${value}`)
    try {
      await sendTransaction(txParams)
      break
    } catch (e) {
      logger.error('Failed to send transaction')
      throw e
    }
  }
}

export async function reportData({
  value,
  aggregator,
  logger
}: {
  value: bigint
  aggregator: IAggregator
  logger: Logger
}) {
  if (await shouldReport({ aggregator, value, logger })) {
    await submit({ value, oracleAddress: aggregator.address, logger })
  }
}
