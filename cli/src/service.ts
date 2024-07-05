import axios from 'axios'
import { command, option, string as cmdstring, subcommands } from 'cmd-ts'
import { ORAKL_NETWORK_API_URL } from './settings.js'
import { buildUrl, idOption, isOraklNetworkApiHealthy } from './utils.js'

const SERVICE_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'service')

export function serviceSub() {
  // service list
  // service insert --name ${name}
  // service remove --id ${id}

  const list = command({
    name: 'list',
    args: {},
    handler: listHandler(true),
  })

  const insert = command({
    name: 'insert',
    args: {
      name: option({
        type: cmdstring,
        long: 'name',
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

  return subcommands({
    name: 'service',
    cmds: { list, insert, remove },
  })
}

export function listHandler(print?: boolean) {
  async function wrapper() {
    if (!(await isOraklNetworkApiHealthy())) return

    try {
      const result = (await axios.get(SERVICE_ENDPOINT))?.data
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
      const response = (await axios.post(SERVICE_ENDPOINT, { name }))?.data
      console.dir(response, { depth: null })
      return response
    } catch (e) {
      console.error('Service was not inserted. Reason:')
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
      const endpoint = buildUrl(SERVICE_ENDPOINT, id.toString())
      const result = (await axios.delete(endpoint))?.data
      console.dir(result, { depth: null })
    } catch (e) {
      console.error('Service was not deleted. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}
