import { command, option, string as cmdstring, subcommands } from 'cmd-ts'
import { insertHandler as adapterInsertHandler } from './adapter'
import { insertHandler as aggregatorInsertHandler } from './aggregator'
import { ReadFile } from './cli-types'
import {
  contractConnectHandler,
  contractInsertHandler,
  functionInsertHandler,
  reporterInsertHandler as delegatorReporterInsertHandler
} from './delegator'
import {
  activateHandler as activateListenerHandler,
  insertHandler as listenerInsertHandler,
  listHandler as listenerListHandler
} from './listener'
import {
  activateHandler as activateReporterHandler,
  insertHandler as reporterInsertHandler,
  listHandler as reporterListHandler
} from './reporter'
import { isValidUrl, loadJsonFromUrl } from './utils'

import { activateHandler as activateAggregatorHandler } from './aggregator'
import { startHandler as startFetcherHandler } from './fetcher'

interface InsertElement {
  adapterSource: string
  aggregatorSource: string
  reporter: {
    walletAddress: string
    walletPrivateKey: string
  }
}

export function datafeedSub() {
  // datafeed bulk-insert --source ${source}
  // datafeed bulk-activate --source ${source}

  const insert = command({
    name: 'bulk-insert',
    args: {
      data: option({
        type: ReadFile,
        long: 'source'
      }),
      chain: option({
        type: cmdstring,
        long: 'chain'
      })
    },
    handler: bulkInsertHandler()
  })

  const activate = command({
    name: 'bulk-activate',
    args: {
      data: option({
        type: ReadFile,
        long: 'source'
      }),
      chain: option({
        type: cmdstring,
        long: 'chain'
      })
    },
    handler: bulkActivateHandler()
  })

  return subcommands({
    name: 'adapter',
    cmds: { insert, activate }
  })
}

export function bulkInsertHandler() {
  async function wrapper({ data, chain }: { data; chain: string }) {
    if (!checkBulkSource(data)) {
      console.error('invalid json src format')
      return
    }
    for (const insertElement of data) {
      const adapterData = await loadJsonFromUrl(insertElement.adapterSource)
      const aggregatorData = await loadJsonFromUrl(insertElement.aggregatorSource)

      await adapterInsertHandler()({ data: adapterData })
      await aggregatorInsertHandler()({ data: aggregatorData, chain })

      const reporterInsertResult = await delegatorReporterInsertHandler()({
        address: insertElement.reporter.walletAddress,
        organizationId: 1 // bisonai fixed to 1
      })
      const contractInsertResult = await contractInsertHandler()({
        address: aggregatorData.address
      })
      await functionInsertHandler()({
        name: 'submit(uint256, int256)',
        contractId: contractInsertResult.id
      })
      await contractConnectHandler()({
        contractId: contractInsertResult.id,
        reporterId: reporterInsertResult.id
      })

      await reporterInsertHandler()({
        chain,
        service: 'DATA_FEED',
        privateKey: insertElement.reporter.walletPrivateKey,
        address: insertElement.reporter.walletAddress,
        oracleAddress: aggregatorData.address
      })

      await listenerInsertHandler()({
        chain,
        service: 'DATA_FEED',
        address: aggregatorData.address,
        eventName: 'NewRound'
      })
    }
  }
  return wrapper
}

export function bulkActivateHandler() {
  const FETCHER_PORT = 4040
  const WORKER_PORT = 5000
  const LISTENER_PORT = 4000
  const REPORTER_PORT = 6000
  async function wrapper({ data, chain }: { data; chain: string }) {
    const listeners = await listenerListHandler(false)({ chain, service: 'DATA_FEED' })
    const reporters = await reporterListHandler(false)({ chain, service: 'DATA_FEED' })

    for (const insertElement of data) {
      const aggregatorData = await loadJsonFromUrl(insertElement.aggregatorSource)

      const reporterId = reporters.find(
        (item) => item.oracleAddress == aggregatorData.contractAddress
      )
      const listenerId = listeners.find((item) => item.address == aggregatorData.address)

      await startFetcherHandler()({
        id: aggregatorData.aggregatorHash,
        chain,
        host: 'http://fetcher.orakl.svc.cluster.local',
        port: '4040'
      })
      await activateAggregatorHandler()({
        aggregatorHash: aggregatorData.aggregatorHash,
        host: 'http://worker.orakl.svc.cluster.local',
        port: '5000'
      })

      await activateReporterHandler()({
        id: reporterId,
        host: 'http://reporter.orakl.svc.cluster.local',
        port: REPORTER_PORT.toString()
      })
      await activateListenerHandler()({
        id: listenerId,
        host: 'http://listener.orakl.svc.cluster.local',
        port: LISTENER_PORT.toString()
      })
    }
  }
  return wrapper
}

function checkBulkSource(data: InsertElement[]) {
  if (!data || data.length == 0) {
    return false
  }
  for (const insertElement of data) {
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
