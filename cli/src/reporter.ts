import axios from 'axios'
import { command, option, string as cmdstring, subcommands } from 'cmd-ts'
import {
  buildUrl,
  chainOptionalOption,
  idOption,
  isOraklNetworkApiHealthy,
  isServiceHealthy,
  serviceOptionalOption,
} from './utils.js'

import { ORAKL_NETWORK_API_URL, REPORTER_SERVICE_HOST, REPORTER_SERVICE_PORT } from './settings.js'

const REPORTER_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'reporter')
const AGGREGATOR_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'aggregator')

export function reporterSub() {
  // reporter list   [--chain ${chain}] [--service ${service}]
  // reporter insert  --chain ${chain}   --service ${service} --address ${address} --privateKey ${privateKey} --oracleAddress ${oracleAddress}
  // reporter remove  --id ${id}
  // reporter active --host ${host} --port ${port}
  // reporter activate --host ${host} --port ${port} --id ${id}
  // reporter deactivate --host ${host} --port ${port} --id ${id}
  // reporter refresh --host ${host} --port ${port}

  const list = command({
    name: 'list',
    args: {
      chain: chainOptionalOption,
      service: serviceOptionalOption,
    },
    handler: listHandler(true),
  })

  const insert = command({
    name: 'insert',
    args: {
      chain: option({
        type: cmdstring,
        long: 'chain',
      }),
      service: option({
        type: cmdstring,
        long: 'service',
      }),
      address: option({
        type: cmdstring,
        long: 'address',
      }),
      privateKey: option({
        type: cmdstring,
        long: 'privateKey',
      }),
      oracleAddress: option({
        type: cmdstring,
        long: 'oracleAddress',
      }),
    },
    handler: insertHandler(),
  })

  const remove = command({
    name: 'remove',
    args: {
      id: idOption,
    },
    handler: removeHandler(),
  })

  const active = command({
    name: 'active',
    args: {
      host: option({
        type: cmdstring,
        long: 'host',
        defaultValue: () => REPORTER_SERVICE_HOST,
      }),
      port: option({
        type: cmdstring,
        long: 'port',
        defaultValue: () => String(REPORTER_SERVICE_PORT),
      }),
    },
    handler: activeHandler(),
  })

  const activate = command({
    name: 'activate',
    args: {
      id: idOption,
      host: option({
        type: cmdstring,
        long: 'host',
        defaultValue: () => REPORTER_SERVICE_HOST,
      }),
      port: option({
        type: cmdstring,
        long: 'port',
        defaultValue: () => String(REPORTER_SERVICE_PORT),
      }),
    },
    handler: activateHandler(),
  })

  const deactivate = command({
    name: 'deactivate',
    args: {
      id: idOption,
      host: option({
        type: cmdstring,
        long: 'host',
        defaultValue: () => REPORTER_SERVICE_HOST,
      }),
      port: option({
        type: cmdstring,
        long: 'port',
        defaultValue: () => String(REPORTER_SERVICE_PORT),
      }),
    },
    handler: deactivateHandler(),
  })

  const refresh = command({
    name: 'refresh',
    args: {
      host: option({
        type: cmdstring,
        long: 'host',
        defaultValue: () => REPORTER_SERVICE_HOST,
      }),
      port: option({
        type: cmdstring,
        long: 'port',
        defaultValue: () => String(REPORTER_SERVICE_PORT),
      }),
    },
    handler: refreshHandler(),
  })

  return subcommands({
    name: 'reporter',
    cmds: { list, insert, remove, active, activate, deactivate, refresh },
  })
}

export function listHandler(print?: boolean) {
  async function wrapper({ chain, service }: { chain?: string; service?: string }) {
    if (!(await isOraklNetworkApiHealthy())) return

    try {
      const result = (await axios.get(REPORTER_ENDPOINT, { data: { chain, service } }))?.data

      const printResult: any[] = []
      const aggregatorUrl = new URL(AGGREGATOR_ENDPOINT)
      const aggregatorResult = (await axios.get(aggregatorUrl.toString())).data
      if (print) {
        for (const reporter of result) {
          if (reporter.service != 'DATA_FEED') {
            printResult.push({ ...reporter })
            continue
          }

          const aggregator = aggregatorResult.find(
            (aggregator) => aggregator.address === reporter.oracleAddress,
          )
          if (aggregator) {
            printResult.push({ ...reporter, name: aggregator.name })
          } else {
            printResult.push({ ...reporter })
          }
        }

        console.dir(printResult, { depth: null })
      }
      return result
    } catch (e) {
      console.dir(e?.response?.data, { depth: null })
    }
  }
  return wrapper
}

export function insertHandler() {
  async function wrapper({
    chain,
    service,
    address,
    privateKey,
    oracleAddress,
  }: {
    chain: string
    service: string
    address: string
    privateKey: string
    oracleAddress: string
  }) {
    if (!(await isOraklNetworkApiHealthy())) return

    try {
      const result = (
        await axios.post(REPORTER_ENDPOINT, { chain, service, address, privateKey, oracleAddress })
      ).data
      console.dir(result, { depth: null })
    } catch (e) {
      console.error('Reporter was not inserted. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function removeHandler() {
  async function wrapper({ id }: { id: number }) {
    if (!(await isOraklNetworkApiHealthy())) return

    const endpoint = buildUrl(REPORTER_ENDPOINT, id.toString())

    try {
      const result = (await axios.delete(endpoint)).data
      console.dir(result, { depth: null })
    } catch (e) {
      console.error('Reporter was not deleted. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function activeHandler() {
  async function wrapper({ host, port }: { host: string; port: string }) {
    const reporterServiceEndpoint = `${host}:${port}`
    if (!(await isServiceHealthy(reporterServiceEndpoint))) return

    const activeReporterEndpoint = buildUrl(reporterServiceEndpoint, 'active')

    try {
      const result = (await axios.get(activeReporterEndpoint)).data
      console.log(result)
    } catch (e) {
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function activateHandler() {
  async function wrapper({ host, port, id }: { host: string; port: string; id: number }) {
    const reporterServiceEndpoint = `${host}:${port}`
    if (!(await isServiceHealthy(reporterServiceEndpoint))) return

    const activateReporterEndpoint = buildUrl(reporterServiceEndpoint, `activate/${id}`)

    try {
      const result = (await axios.get(activateReporterEndpoint)).data
      console.log(result?.message)
    } catch (e) {
      console.error('Reporter was not activated. Reason:')
      console.error(e?.response?.data?.message)
      throw e
    }
  }
  return wrapper
}

export function deactivateHandler() {
  async function wrapper({ host, port, id }: { host: string; port: string; id: number }) {
    const reporterServiceEndpoint = `${host}:${port}`
    if (!(await isServiceHealthy(reporterServiceEndpoint))) return

    const deactivateReporterEndpoint = buildUrl(reporterServiceEndpoint, `deactivate/${id}`)

    try {
      const result = (await axios.get(deactivateReporterEndpoint)).data
      console.log(result?.message)
    } catch (e) {
      console.error('Reporter was not deactivated. Reason:')
      console.error(e?.response?.data?.message)
      throw e
    }
  }
  return wrapper
}

export function refreshHandler() {
  async function wrapper({ host, port }: { host: string; port: string }) {
    const reporterServiceEndpoint = `${host}:${port}`
    if (!(await isServiceHealthy(reporterServiceEndpoint))) return

    const refreshReporterEndpoint = buildUrl(reporterServiceEndpoint, 'refresh')

    try {
      const result = (await axios.get(refreshReporterEndpoint)).data
      console.log(result?.message)
    } catch (e) {
      console.error('Reporters were not refreshed. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}
