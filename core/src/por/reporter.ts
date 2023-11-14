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
import { IAggregator, IReporterConfig } from '../types'
import { buildTransaction } from '../worker/data-feed.utils'

async function shouldReport({ aggregator, value }: { aggregator: IAggregator; value: bigint }) {
  const contract = new ethers.Contract(aggregator.address, Aggregator__factory.abi, PROVIDER)
  const latestRoundData = await contract.latestRoundData()

  // Check Submission Hearbeat
  const updatedAt = Number(latestRoundData.updatedAt) * 1000 // convert to milliseconds
  const timestamp = Date.now()
  const heartbeat = aggregator.heartbeat

  if (updatedAt + heartbeat < timestamp) {
    return true
  } else if (aggregator.threshold && latestRoundData.answer) {
    // Check Deviation  Threashold
    const latestSubmission = Number(latestRoundData.answer)
    const currentSubmission = Number(value)

    const range = latestSubmission * aggregator.threshold
    const l = currentSubmission - range
    const r = currentSubmission + range
    return currentSubmission < l || currentSubmission > r
  }
  return false
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
  const contract = new ethers.Contract(oracleAddress, Aggregator__factory.abi, PROVIDER)
  let queriedRoundId = 0
  const state = await contract.oracleRoundState(reporter.address, queriedRoundId)
  const roundId = state._roundId
  const getLastestRoundData = await contract.getRoundData(roundId - 1)

  console.log('GetLastestRoundData from report:', getLastestRoundData)
  console.log('RoundId: ', roundId)
  console.log('Sumbission: ', value)

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
  aggregator,
  logger
}: {
  value: bigint
  aggregator: IAggregator
  logger: Logger
}) {
  if (await shouldReport({ aggregator, value })) {
    await report({ value, oracleAddress: aggregator.address, logger })
  }
}
