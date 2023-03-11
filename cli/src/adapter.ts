import axios from 'axios'
import { command, flag, subcommands, option, boolean as cmdboolean } from 'cmd-ts'
import { idOption, buildUrl, computeAdapterHash } from './utils'
import { ReadFile, IAdapter } from './cli-types'
import { ORAKL_NETWORK_API_URL } from './settings'

const ADAPTER_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'api/v1/adapter')

export function adapterSub() {
  // adapter list
  // adapter insert --file-path [file-path]
  // adapter remove --id [id]
  // adapter hash --file-path [file-path] --verify

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

  const hash = command({
    name: 'hash',
    args: {
      verify: flag({
        type: cmdboolean,
        long: 'verify'
      }),
      data: option({
        type: ReadFile,
        long: 'file-path'
      })
    },
    handler: hashHandler()
  })

  return subcommands({
    name: 'adapter',
    cmds: { list, insert, remove, hash }
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

export function hashHandler() {
  async function wrapper({ data, verify }: { data; verify: boolean }) {
    try {
      const adapter = data as IAdapter
      const adapterWithCorrectHash = await computeAdapterHash({ data: adapter, verify })
      console.dir(adapterWithCorrectHash, { depth: null })
    } catch (e) {
      console.error(e.message)
    }
  }
  return wrapper
}
