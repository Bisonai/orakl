import axios from 'axios'
import { command, subcommands, option, string as cmdstring } from 'cmd-ts'
import { idOption, buildUrl, isOraklNetworkApiHealthy } from './utils'
import { ORAKL_NETWORK_API_URL } from './settings'

const CHAIN_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'api/v1/chain')

export function chainSub() {
  // chain list
  // chain insert --name [name]
  // chain remove --id [id]

  const list = command({
    name: 'list',
    args: {},
    handler: listHandler(true)
  })

  const insert = command({
    name: 'insert',
    args: {
      name: option({
        type: cmdstring,
        long: 'name'
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
    name: 'chain',
    cmds: { list, insert, remove }
  })
}

export function listHandler(print?: boolean) {
  async function wrapper() {
    if (!(await isOraklNetworkApiHealthy())) return

    try {
      const result = (await axios.get(CHAIN_ENDPOINT))?.data
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
  async function wrapper({ name }: { name: string }) {
    if (!(await isOraklNetworkApiHealthy())) return

    try {
      const response = (await axios.post(CHAIN_ENDPOINT, { name }))?.data
      console.dir(response, { depth: null })
    } catch (e) {
      console.dir(e?.response?.data, { depth: null })
    }
  }
  return wrapper
}

export function removeHandler() {
  async function wrapper({ id }: { id: number }) {
    if (!(await isOraklNetworkApiHealthy())) return

    try {
      const endpoint = buildUrl(CHAIN_ENDPOINT, id.toString())
      const result = (await axios.delete(endpoint))?.data
      console.dir(result, { depth: null })
    } catch (e) {
      console.dir(e?.response?.data, { depth: null })
    }
  }
  return wrapper
}
