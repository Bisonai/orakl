import axios from 'axios'
import { command, subcommands, option, string as cmdstring } from 'cmd-ts'
import {
  idOption,
  chainOptionalOption,
  serviceOptionalOption,
  buildUrl,
  isOraklNetworkApiHealthy
} from './utils'

import { ORAKL_NETWORK_API_URL } from './settings'

const LISTENER_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'listener')

export function listenerSub() {
  // listener list   [--chain ${chain}] [--service ${service}]
  // listener insert  --chain ${chain}   --service ${service} --address ${address} --eventName ${eventName}
  // listener remove  --id ${id}

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
      eventName: option({
        type: cmdstring,
        long: 'eventName'
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
    name: 'listener',
    cmds: { list, insert, remove }
  })
}

export function listHandler(print?: boolean) {
  async function wrapper({ chain, service }: { chain?: string; service?: string }) {
    if (!(await isOraklNetworkApiHealthy())) return

    try {
      const result = (await axios.get(LISTENER_ENDPOINT, { data: { chain, service } }))?.data
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
    eventName
  }: {
    chain: string
    service: string
    address: string
    eventName: string
  }) {
    if (!(await isOraklNetworkApiHealthy())) return

    try {
      const result = (await axios.post(LISTENER_ENDPOINT, { chain, service, address, eventName }))
        .data
      console.dir(result, { depth: null })
    } catch (e) {
      console.error('Listener was not inserted. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function removeHandler() {
  async function wrapper({ id }: { id: number }) {
    if (!(await isOraklNetworkApiHealthy())) return

    const endpoint = buildUrl(LISTENER_ENDPOINT, id.toString())

    try {
      const result = (await axios.delete(endpoint)).data
      console.dir(result, { depth: null })
    } catch (e) {
      console.error('Listener was not deleted. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}
