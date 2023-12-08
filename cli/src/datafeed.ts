import { command, option, subcommands } from 'cmd-ts'
import { insertHandler as adapterInsertHandler } from './adapter'
import { insertHandler as aggregatorInsertHandler } from './aggregator'
import { IDatafeedBulk, IDatafeedBulkInsertElement, ReadFile } from './cli-types'
import {
  contractConnectHandler,
  contractInsertHandler,
  functionInsertHandler,
  organizationListHandler,
  reporterInsertHandler as delegatorReporterInsertHandler
} from './delegator'
import { insertHandler as listenerInsertHandler } from './listener'
import { insertHandler as reporterInsertHandler } from './reporter'
import { isValidUrl, loadJsonFromUrl } from './utils'

export function datafeedSub() {
  // datafeed bulk-insert --source ${source}

  // TODOs
  // datafeed bulk-remove --source ${source}
  // datafeed bulk-activate --source ${source}
  // datafeed bulk-deactivate --source ${source}

  const insert = command({
    name: 'bulk-insert',
    args: {
      data: option({
        type: ReadFile,
        long: 'source'
      })
    },
    handler: bulkInsertHandler()
  })

  return subcommands({
    name: 'datafeed',
    cmds: { insert }
  })
}

export function bulkInsertHandler() {
  async function wrapper({ data }: { data }) {
    const bulkData = data as IDatafeedBulk

    const chain = bulkData?.chain || 'localhost'
    const service = bulkData?.service || 'DATA_FEED'
    const organization = bulkData?.organization || 'bisonai'
    const functionName = bulkData?.functionName || 'submit(uint256, int256)'
    const eventName = bulkData?.eventName || 'NewRound'
    const organizationId = (await organizationListHandler()()).find(
      (_organization) => _organization.name == organization
    ).id

    if (!checkBulkSource(data?.bulk)) {
      console.error('invalid json src format')
      return
    }
    for (const insertElement of bulkData.bulk) {
      const adapterData = await loadJsonFromUrl(insertElement.adapterSource)
      const aggregatorData = await loadJsonFromUrl(insertElement.aggregatorSource)

      await adapterInsertHandler()({ data: adapterData })
      await aggregatorInsertHandler()({ data: aggregatorData, chain })

      const reporterInsertResult = await delegatorReporterInsertHandler()({
        address: insertElement.reporter.walletAddress,
        organizationId: Number(organizationId)
      })
      const contractInsertResult = await contractInsertHandler()({
        address: aggregatorData.address
      })

      await functionInsertHandler()({
        name: functionName,
        contractId: Number(contractInsertResult.id)
      })
      await contractConnectHandler()({
        contractId: Number(contractInsertResult.id),
        reporterId: Number(reporterInsertResult.id)
      })
      await reporterInsertHandler()({
        chain,
        service: service,
        privateKey: insertElement.reporter.walletPrivateKey,
        address: insertElement.reporter.walletAddress,
        oracleAddress: aggregatorData.address
      })
      await listenerInsertHandler()({
        chain,
        service: service,
        address: aggregatorData.address,
        eventName
      })
    }
  }
  return wrapper
}

function checkBulkSource(bulkData: IDatafeedBulkInsertElement[]) {
  if (!bulkData || bulkData.length == 0) {
    return false
  }
  for (const insertElement of bulkData) {
    if (!isValidUrl(insertElement.adapterSource) || !isValidUrl(insertElement.aggregatorSource)) {
      return false
    }
    if (
      !insertElement.reporter ||
      !insertElement.reporter.walletAddress ||
      !insertElement.reporter.walletPrivateKey
    ) {
      return false
    }
  }
  return true
}
