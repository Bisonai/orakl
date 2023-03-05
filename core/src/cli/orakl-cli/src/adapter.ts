import axios from 'axios'
import { flag, command, subcommands, option, string as cmdstring } from 'cmd-ts'
import {
  chainOptionalOption,
  chainToId,
  dryrunOption,
  idOption,
  formatResultInsert,
  formatResultRemove,
  buildUrl
} from './utils'
import { computeDataHash } from './utils'
import { ReadFile } from './cli-types'
import { ORAKL_NETWORK_API_URL } from './settings'

const ADAPTER_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'adapter')

export function adapterSub(db) {
  // adapter list
  // adapter insert --file-path [file-path]
  // adapter remove --id [id]

  const list = command({
    name: 'list',
    args: {
      chain: chainOptionalOption
    },
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
      id: idOption,
      dryrun: dryrunOption
    },
    handler: removeHandler()
  })

  return subcommands({
    name: 'adapter',
    cmds: { list, insert, remove }
  })
}

export function listHandler(print?: boolean) {
  async function wrapper({ chain }: { chain?: string }) {
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
    const result = (await axios.post(ADAPTER_ENDPOINT, data)).data
    console.dir(result, { depth: null })
  }
  return wrapper
}

export function removeHandler() {
  async function wrapper({ id }: { id: number }) {
    const endpoint = buildUrl(ADAPTER_ENDPOINT, id.toString())
    const result = (await axios.delete(endpoint)).data
    console.dir(result, { depth: null })
  }
  return wrapper
}
