import axios from 'axios'
import { command, subcommands, option } from 'cmd-ts'
import { idOption, buildUrl } from './utils'
import { ReadFile } from './cli-types'
import { ORAKL_NETWORK_API_URL } from './settings'

const ADAPTER_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'adapter')

export function adapterSub() {
  // adapter list
  // adapter insert --file-path [file-path]
  // adapter remove --id [id]

  const list = command({
    name: 'list',
    args: {},
    handler: listHandler(true)
  })

  const insert = command({
    name: 'insert',
    args: {
      data: option({
        type: ReadFile,
        long: 'file-path'
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
    name: 'adapter',
    cmds: { list, insert, remove }
  })
}

export function listHandler(print?: boolean) {
  async function wrapper() {
    const result = (await axios.get(ADAPTER_ENDPOINT)).data
    if (print) {
      console.dir(result, { depth: null })
    }
    return result
  }
  return wrapper
}

export function insertHandler() {
  async function wrapper({ data }: { data }) {
    try {
      const response = (await axios.post(ADAPTER_ENDPOINT, data)).data
      console.dir(response, { depth: null })
    } catch (e) {
      console.dir(e?.response?.data, { depth: null })
    }
  }
  return wrapper
}

export function removeHandler() {
  async function wrapper({ id }: { id: number }) {
    try {
      const endpoint = buildUrl(ADAPTER_ENDPOINT, id.toString())
      const response = (await axios.delete(endpoint)).data
      console.dir(response, { depth: null })
    } catch (e) {
      console.dir(e?.response?.data, { depth: null })
    }
  }
  return wrapper
}
