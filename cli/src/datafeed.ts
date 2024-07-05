import { command, option, subcommands } from 'cmd-ts'
import { insertHandler as adapterInsertHandler } from './adapter.js'
import {
  activateHandler as aggregatorActivateHandler,
  deactivateHandler as aggregatorDeactivateHandler,
  insertHandler as aggregatorInsertHandler,
} from './aggregator.js'
import {
  IAdapter,
  IAggregator,
  IDatafeedBulk,
  IDatafeedBulkInsertElement,
  ReadFile,
  readFileFromSource,
} from './cli-types.js'
import {
  contractConnectHandler,
  contractInsertHandler,
  contractListHandler,
  contractRemoveHandler,
  functionInsertHandler,
  functionListHandler,
  functionRemoveHandler,
  organizationListHandler,
  reporterInsertHandler as delegatorReporterInsertHandler,
  reporterListHandler as delegatorReporterListHandler,
  reporterRemoveHandler as delegatorReporterRemoveHandler,
} from './delegator.js'
import {
  startHandler as fetcherStartHandler,
  stopHandler as fetcherStopHandler,
} from './fetcher.js'
import {
  activateHandler as listenerActivateHandler,
  deactivateHandler as listenerDeactivateHandler,
  insertHandler as listenerInsertHandler,
  listHandler as listenerListHandler,
  removeHandler as listenerRemoveHandler,
} from './listener.js'
import {
  activateHandler as reporterActivateHandler,
  deactivateHandler as reporterDeactivateHandler,
  insertHandler as reporterInsertHandler,
  listHandler as reporterListHandler,
  removeHandler as reporterRemoveHandler,
} from './reporter.js'
import {
  FETCHER_HOST,
  FETCHER_PORT,
  LISTENER_SERVICE_HOST,
  LISTENER_SERVICE_PORT,
  REPORTER_SERVICE_HOST,
  REPORTER_SERVICE_PORT,
  WORKER_SERVICE_HOST,
  WORKER_SERVICE_PORT,
} from './settings.js'
import { isValidUrl } from './utils.js'

export function datafeedSub() {
  // datafeed bulk-insert --source ${source}
  // datafeed bulk-remove --source ${source}
  // datafeed bulk-activate --source ${source}
  // datafeed bulk-deactivate --source ${source}

  const insert = command({
    name: 'bulk-insert',
    args: {
      data: option({
        type: ReadFile,
        long: 'source',
      }),
    },
    handler: bulkInsertHandler(),
  })

  const remove = command({
    name: 'bulk-remove',
    args: {
      data: option({
        type: ReadFile,
        long: 'source',
      }),
    },
    handler: bulkRemoveHandler(),
  })

  const activate = command({
    name: 'bulk-activate',
    args: {
      data: option({
        type: ReadFile,
        long: 'source',
      }),
    },
    handler: bulkActivateHandler(),
  })

  const deactivate = command({
    name: 'bulk-deactivate',
    args: {
      data: option({
        type: ReadFile,
        long: 'source',
      }),
    },
    handler: bulkDeactivateHandler(),
  })

  return subcommands({
    name: 'datafeed',
    cmds: { insert, remove, activate, deactivate },
  })
}

export function bulkInsertHandler() {
  async function wrapper({ data }: { data }) {
    const bulkData = data as IDatafeedBulk

    const chain = bulkData?.chain || 'localhost'
    const service = bulkData?.service || 'DATA_FEED'
    const organization = bulkData?.organization || 'bisonai'
    const functionName = bulkData?.functionName || 'submit(uint256,int256)'
    const eventName = bulkData?.eventName || 'NewRound'
    const organizationId = (await organizationListHandler()()).find(
      (_organization) => _organization.name == organization,
    ).id

    if (!checkBulkSource(data?.bulk)) {
      console.error('invalid json src format')
      return
    }

    for (const insertElement of bulkData.bulk) {
      console.log(`inserting ${insertElement}`)
      const adapterData = await readFileFromSource(insertElement.adapterSource)
      if (!checkAdapterSource(adapterData)) {
        console.error(`invalid adapterData: ${adapterData}, skipping insert`)
        continue
      }
      const aggregatorData = await readFileFromSource(insertElement.aggregatorSource)
      if (!checkAggregatorSource(aggregatorData)) {
        console.error(`invalid aggregatorData: ${aggregatorData}, skipping insert`)
        continue
      }

      await adapterInsertHandler()({ data: adapterData })
      await aggregatorInsertHandler()({ data: aggregatorData, chain })

      const reporterInsertResult = await delegatorReporterInsertHandler()({
        address: insertElement.reporter.walletAddress,
        organizationId: Number(organizationId),
      })
      const contractInsertResult = await contractInsertHandler()({
        address: aggregatorData.address,
      })

      await functionInsertHandler()({
        name: functionName,
        contractId: Number(contractInsertResult.id),
      })
      await contractConnectHandler()({
        contractId: Number(contractInsertResult.id),
        reporterId: Number(reporterInsertResult.id),
      })
      await reporterInsertHandler()({
        chain,
        service: service,
        privateKey: insertElement.reporter.walletPrivateKey,
        address: insertElement.reporter.walletAddress,
        oracleAddress: aggregatorData.address,
      })
      await listenerInsertHandler()({
        chain,
        service: service,
        address: aggregatorData.address,
        eventName,
      })
    }
  }
  return wrapper
}

export function bulkRemoveHandler() {
  async function wrapper({ data }: { data }) {
    const bulkData = data as IDatafeedBulk

    if (!checkBulkSource(data?.bulk)) {
      console.error('invalid json src format')
      return
    }

    const listeners = await listenerListHandler()({})
    const reporters = await reporterListHandler()({})
    const delegatorReporters = await delegatorReporterListHandler()()
    const delegatorContracts = await contractListHandler()()
    const delegatorFunctions = await functionListHandler()()

    for (const removeElement of bulkData.bulk) {
      console.log(`removing ${removeElement}`)
      const adapterData = await readFileFromSource(removeElement.adapterSource)
      if (!checkAdapterSource(adapterData)) {
        console.error(`invalid adapterData: ${adapterData}, skipping removal`)
        continue
      }
      const aggregatorData = await readFileFromSource(removeElement.aggregatorSource)
      if (!checkAggregatorSource(aggregatorData)) {
        console.error(`invalid aggregatorData: ${aggregatorData}, skipping removal`)
        continue
      }

      const listenerId = listeners.find((listener) => listener.address == aggregatorData.address).id
      const reporterId = reporters.find(
        (reporter) => reporter.address == removeElement.reporter.walletAddress,
      ).id

      const delegatorReporterId = delegatorReporters.find(
        (reporter) =>
          reporter.address.toLowerCase() == removeElement.reporter.walletAddress.toLowerCase(),
      ).id
      const delegatorContractId = delegatorContracts.find(
        (contract) => contract.address.toLowerCase() == aggregatorData.address.toLowerCase(),
      ).id
      const functionId = delegatorFunctions.find(
        (_function) => _function.address.toLowerCase() == aggregatorData.address.toLowerCase(),
      ).id

      await listenerRemoveHandler()({ id: listenerId })
      await reporterRemoveHandler()({ id: reporterId })

      await functionRemoveHandler()({ id: functionId })
      await contractRemoveHandler()({ id: delegatorContractId })
      await delegatorReporterRemoveHandler()({ id: delegatorReporterId })

      // not removing adapter and aggregator since it's impossible to remove those without wiping out related rows from data table
      // and leaving out adapter and aggregator in the table won't have that much impact on db so it'll be as it is
    }
  }
  return wrapper
}

export function bulkActivateHandler() {
  async function wrapper({ data }: { data }) {
    const bulkData = data as IDatafeedBulk

    const chain = bulkData?.chain || 'localhost'
    const service = bulkData?.service || 'DATA_FEED'

    const fetcherHost = bulkData?.fetcherHost || FETCHER_HOST
    const workerHost = bulkData?.workerHost || WORKER_SERVICE_HOST
    const listenerHost = bulkData?.listenerHost || LISTENER_SERVICE_HOST
    const reporterHost = bulkData?.reporterHost || REPORTER_SERVICE_HOST

    const fetcherPort = bulkData?.fetcherPort || FETCHER_PORT
    const workerPort = bulkData?.workerPort || WORKER_SERVICE_PORT
    const listenerPort = bulkData?.listenerPort || LISTENER_SERVICE_PORT
    const reporterPort = bulkData?.reporterPort || REPORTER_SERVICE_PORT

    const listeners = await listenerListHandler()({ chain, service })
    const reporters = await reporterListHandler()({ chain, service })

    if (!checkBulkSource(data?.bulk)) {
      console.error(`invalid json src format`)
      return
    }

    for (const activateElement of data.bulk) {
      const aggregatorData = await readFileFromSource(activateElement.aggregatorSource)
      if (!checkAggregatorSource(aggregatorData)) {
        console.error(`invalid aggregatorData: ${aggregatorData}, skipping activation`)
        continue
      }

      const reporterId = reporters.find(
        (reporter) =>
          reporter.address.toLowerCase() == activateElement.reporter.walletAddress.toLowerCase(),
      )?.id
      if (!reporterId) {
        console.error(
          `reporterId not found for ${activateElement.reporter.walletAddress}, skipping activation`,
        )
        continue
      }

      const listenerId = listeners.find(
        (listener) => listener.address == aggregatorData.address,
      )?.id
      if (!listenerId) {
        console.error(`listenerId not found for ${aggregatorData.address}, skipping activation`)
        continue
      }

      try {
        await fetcherStartHandler()({
          id: aggregatorData.aggregatorHash,
          chain,
          host: fetcherHost,
          port: fetcherPort,
        })
        await aggregatorActivateHandler()({
          aggregatorHash: aggregatorData.aggregatorHash,
          host: workerHost,
          port: workerPort,
        })

        await reporterActivateHandler()({
          id: Number(reporterId),
          host: reporterHost,
          port: reporterPort,
        })
        await listenerActivateHandler()({
          id: Number(listenerId),
          host: listenerHost,
          port: listenerPort,
        })
      } catch (e) {
        console.error(
          `activation failed for ${activateElement.aggregatorSource}, breaking iteration`,
        )
        console.error(e?.response?.data)
        break
      }
    }
  }
  return wrapper
}

export function bulkDeactivateHandler() {
  async function wrapper({ data }: { data }) {
    const bulkData = data as IDatafeedBulk

    const chain = bulkData?.chain || 'localhost'
    const service = bulkData?.service || 'DATA_FEED'

    const fetcherHost = bulkData?.fetcherHost || FETCHER_HOST
    const workerHost = bulkData?.workerHost || WORKER_SERVICE_HOST
    const listenerHost = bulkData?.listenerHost || LISTENER_SERVICE_HOST
    const reporterHost = bulkData?.reporterHost || REPORTER_SERVICE_HOST

    const fetcherPort = bulkData?.fetcherPort || FETCHER_PORT
    const workerPort = bulkData?.workerPort || WORKER_SERVICE_PORT
    const listenerPort = bulkData?.listenerPort || LISTENER_SERVICE_PORT
    const reporterPort = bulkData?.reporterPort || REPORTER_SERVICE_PORT

    const listeners = await listenerListHandler()({ chain, service })
    const reporters = await reporterListHandler()({ chain, service })

    if (!checkBulkSource(data?.bulk)) {
      console.error(`invalid json src format`)
      return
    }

    for (const deactivateElement of data.bulk) {
      const aggregatorData = await readFileFromSource(deactivateElement.aggregatorSource)
      if (!checkAggregatorSource(aggregatorData)) {
        console.error(`invalid aggregatorData: ${aggregatorData}, skipping deactivation`)
        continue
      }

      const reporterId = reporters.find(
        (reporter) =>
          reporter.address.toLowerCase() == deactivateElement.reporter.walletAddress.toLowerCase(),
      )?.id
      if (!reporterId) {
        console.error(
          `reporterId not found for ${deactivateElement.reporter.walletAddress}, skipping deactivation`,
        )
        continue
      }

      const listenerId = listeners.find(
        (listener) => listener.address == aggregatorData.address,
      )?.id
      if (!listenerId) {
        console.error(`listenerId not found for ${aggregatorData.address}, skipping deactivation`)
        continue
      }

      try {
        await listenerDeactivateHandler()({
          id: Number(listenerId),
          host: listenerHost,
          port: listenerPort,
        })

        await reporterDeactivateHandler()({
          id: Number(reporterId),
          host: reporterHost,
          port: reporterPort,
        })

        await aggregatorDeactivateHandler()({
          aggregatorHash: aggregatorData.aggregatorHash,
          host: workerHost,
          port: workerPort,
        })

        await fetcherStopHandler()({
          id: aggregatorData.aggregatorHash,
          chain,
          host: fetcherHost,
          port: fetcherPort,
        })
      } catch (e) {
        console.error(
          `deactivation failed for ${deactivateElement.aggregatorSource}, breaking iteration`,
        )
        console.error(e?.response?.data)
        break
      }
    }
  }
  return wrapper
}

function checkBulkSource(bulkData: IDatafeedBulkInsertElement[]) {
  if (!bulkData || bulkData.length == 0) {
    console.error('empty bulk insert data')
    return false
  }
  for (const insertElement of bulkData) {
    if (!isValidUrl(insertElement.adapterSource)) {
      console.error(`${insertElement.adapterSource} is invalid url`)
      return false
    }
    if (!isValidUrl(insertElement.aggregatorSource)) {
      console.error(`${insertElement.aggregatorSource} is invalid url`)
    }
    if (
      !insertElement.reporter ||
      !insertElement.reporter.walletAddress ||
      !insertElement.reporter.walletPrivateKey
    ) {
      console.error(`${insertElement.reporter} is missing values`)
      return false
    }
  }
  return true
}

function checkAdapterSource(adapterData: IAdapter) {
  if (!adapterData.adapterHash) {
    console.error(`adapterHash is empty`)
    return false
  }
  if (!adapterData.name) {
    console.error(`adapter name is empty`)
    return false
  }
  if (!adapterData.decimals) {
    console.error(`adapter decimals is empty`)
    return false
  }
  if (!adapterData.feeds) {
    console.error(`adapter feeds is empty`)
    return false
  }
  return true
}

function checkAggregatorSource(aggregatorData: IAggregator) {
  if (!aggregatorData.aggregatorHash) {
    console.error(`aggregatorHash is empty`)
    return false
  }
  if (aggregatorData.active) {
    console.error(`can't insert active aggregator`)
    return false
  }
  if (!aggregatorData.name) {
    console.error(`aggregatorData name is empty`)
    return false
  }
  if (!aggregatorData.address) {
    console.error(`aggregator address is empty`)
    return false
  }
  if (!aggregatorData.heartbeat) {
    console.error(`aggregator heartbeat is empty`)
    return false
  }
  if (!aggregatorData.threshold) {
    console.error(`aggregator threshold is empty`)
    return false
  }
  if (!aggregatorData.absoluteThreshold) {
    console.error(`aggregator absoluteThreshold is empty`)
    return false
  }
  if (!aggregatorData.adapterHash) {
    console.error(`aggregator adapterHash is empty`)
    return false
  }
  return true
}
