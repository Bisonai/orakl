import { fetchWithAggregator } from './fetcher'

const aggregatorHash = '0x952f883b8d2fd47a790307cb569118a215ea45eb861cefd4ed3b83ae7550f8e8'

const main = async () => {
  const value = await fetchWithAggregator(aggregatorHash)
  console.log('Value:', value)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
