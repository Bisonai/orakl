import axios from 'axios'
import { command, subcommands, option, string as cmdstring } from 'cmd-ts'
import {
  idOption,
  chainOptionalOption,
  serviceOptionalOption,
  buildUrl,
  isOraklNetworkApiHealthy,
  isServiceHealthy
} from './utils'

import { ORAKL_NETWORK_API_URL, REPORTER_SERVICE_HOST, REPORTER_SERVICE_PORT } from './settings'

const REPORTER_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'reporter')

export function reporterSub() {
  // reporter list   [--chain ${chain}] [--service ${service}]
  // reporter insert  --chain ${chain}   --service ${service} --address ${address} --eventName ${eventName}
  // reporter remove  --id ${id}

  const list = command({
    name: 'list',
    args: {
      chain: chainOptionalOption,
      service: serviceOptionalOption
    },
    handler: listHandler(true)
  })

  const insert = command({
    name: 'insert',
    args: {
      chain: option({
        type: cmdstring,
        long: 'chain'
      }),
      service: option({
        type: cmdstring,
        long: 'service'
      }),
      address: option({
        type: cmdstring,
        long: 'address'
      }),
      privateKey: option({
        type: cmdstring,
        long: 'privateKey'
      }),
      oracleAddress: option({
        type: cmdstring,
        long: 'oracleAddress'
      })
    },
    handler: insertHandler()
  })

  const remove = command({
    name: 'remove',
    args: {
      id: idOption
    },
    handler: removeHandler()
  })

  return subcommands({
    name: 'reporter',
    cmds: { list, insert, remove }
  })
}

export function listHandler(print?: boolean) {
  async function wrapper({ chain, service }: { chain?: string; service?: string }) {
    if (!(await isOraklNetworkApiHealthy())) return

    try {
      const result = (await axios.get(REPORTER_ENDPOINT, { data: { chain, service } }))?.data
      if (print) {
        console.dir(result, { depth: null })
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
    oracleAddress
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
