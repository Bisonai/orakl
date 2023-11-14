import { buildLogger } from '../logger'
import { hookConsoleError } from '../utils'
import { fetchWithAggregator } from './fetcher'
import { reportData } from './reporter'

const aggregatorHash = '0x952f883b8d2fd47a790307cb569118a215ea45eb861cefd4ed3b83ae7550f8e8'
const LOGGER = buildLogger()

const main = async () => {
  hookConsoleError(LOGGER)

  const { value, oracleAddress } = await fetchWithAggregator(aggregatorHash)
  await reportData({ value, oracleAddress, logger: LOGGER })
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
