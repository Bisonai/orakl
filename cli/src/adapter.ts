import axios from 'axios'
import { boolean as cmdboolean, command, flag, option, subcommands } from 'cmd-ts'
import { IAdapter, ReadFile } from './cli-types.js'
import { ORAKL_NETWORK_API_URL } from './settings.js'
import { buildUrl, idOption, isOraklNetworkApiHealthy } from './utils.js'

const ADAPTER_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'adapter')

export function adapterSub() {
  // adapter list
  // adapter insert --source ${source}
  // adapter remove --id ${id}
  // adapter hash --source ${source} --verify

  const list = command({
    name: 'list',
    args: {},
    handler: listHandler(true),
  })

  const insert = command({
    name: 'insert',
    args: {
      data: option({
        type: ReadFile,
        long: 'source',
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

  const hash = command({
    name: 'hash',
    args: {
      verify: flag({
        type: cmdboolean,
        long: 'verify',
      }),
      data: option({
        type: ReadFile,
        long: 'source',
      }),
    },
    handler: hashHandler(),
  })

  return subcommands({
    name: 'adapter',
    cmds: { list, insert, remove, hash },
  })
}

export function listHandler(print?: boolean) {
  async function wrapper() {
    if (!(await isOraklNetworkApiHealthy())) return

    try {
      const result = (await axios.get(ADAPTER_ENDPOINT)).data
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
  async function wrapper({ data }: { data }) {
    if (!(await isOraklNetworkApiHealthy())) return

    try {
      const response = (await axios.post(ADAPTER_ENDPOINT, data)).data
      console.dir(response, { depth: null })
      return response
    } catch (e) {
      console.error('Adapter was not inserted. Reason:')
      console.error(e?.response?.data)
      return e?.response?.data
    }
  }
  return wrapper
}

export function removeHandler() {
  async function wrapper({ id }: { id: number }) {
    if (!(await isOraklNetworkApiHealthy())) return

    try {
      const endpoint = buildUrl(ADAPTER_ENDPOINT, id.toString())
      const response = (await axios.delete(endpoint)).data
      console.dir(response, { depth: null })
    } catch (e) {
      console.error('Adapter was not deleted. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function hashHandler() {
  async function wrapper({ data, verify }: { data; verify: boolean }) {
    try {
      const endpoint = buildUrl(ADAPTER_ENDPOINT, 'hash')
      const adapter = data as IAdapter
      const adapterWithCorrectHash = (await axios.post(endpoint, adapter, { params: { verify } }))
        .data
      console.dir(adapterWithCorrectHash, { depth: null })
      return adapterWithCorrectHash
    } catch (e) {
      console.error('Adapter hash could not be computed. Reason:')
      const errMsg = e?.response?.data ? e.response.data : e.message

      console.error(errMsg)
      return errMsg
    }
  }
  return wrapper
}
