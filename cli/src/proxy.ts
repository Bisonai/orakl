import axios from 'axios'
import { command, subcommands, option, string as cmdstring } from 'cmd-ts'
import { idOption, buildUrl, isOraklNetworkApiHealthy } from './utils'
import { ORAKL_NETWORK_API_URL } from './settings'

const PROXY_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'proxy')

export function proxySub() {
  // proxy list
  // proxy insert --protocol ${protocol} --host ${host} --port ${port}
  // proxy remove --id ${id}

  const list = command({
    name: 'list',
    args: {},
    handler: listHandler(true)
  })

  const insert = command({
    name: 'insert',
    args: {
      protocol: option({
        type: cmdstring,
        long: 'protocol'
      }),
      host: option({
        type: cmdstring,
        long: 'host'
      }),
      port: option({
        type: cmdstring,
        long: 'port'
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
    name: 'proxy',
    cmds: { list, insert, remove }
  })
}

export function listHandler(print?: boolean) {
  async function wrapper() {
    if (!(await isOraklNetworkApiHealthy())) return

    try {
      const result = (await axios.get(PROXY_ENDPOINT))?.data
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
    protocol,
    host,
    port
  }: {
    protocol: string
    host: string
    port: string
  }) {
    if (!(await isOraklNetworkApiHealthy())) return

    try {
      const response = (await axios.post(PROXY_ENDPOINT, { protocol, host, port }))?.data
      console.dir(response, { depth: null })
    } catch (e) {
      console.error('Proxy was not inserted. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function removeHandler() {
  async function wrapper({ id }: { id: number }) {
    if (!(await isOraklNetworkApiHealthy())) return

    try {
      const endpoint = buildUrl(PROXY_ENDPOINT, id.toString())
      const result = (await axios.delete(endpoint))?.data
      console.dir(result, { depth: null })
    } catch (e) {
      console.error('Proxy was not deleted. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}
