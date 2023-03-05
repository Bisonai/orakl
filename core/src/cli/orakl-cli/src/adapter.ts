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
    handler: removeHandler(db)
  })

  const insertFromChain = command({
    name: 'insertFromChain',
    args: {
      adapterId: option({ type: cmdstring, long: 'adapter-id' }),
      fromChain: option({ type: cmdstring, long: 'from-chain' }),
      toChain: option({ type: cmdstring, long: 'to-chain' }),
      dryrun: dryrunOption
    },
    handler: insertFromChainHandler(db)
  })

  return subcommands({
    name: 'adapter',
    cmds: { list, insert, remove, insertFromChain }
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

export function removeHandler(db) {
  async function wrapper({ id, dryrun }: { id: number; dryrun?: boolean }) {
    const query = `DELETE FROM Adapter WHERE id=${id}`
    if (dryrun) {
      console.debug(query)
    } else {
      const result = await db.run(query)
      console.log(formatResultRemove(result))
    }
  }
  return wrapper
}

export function insertFromChainHandler(db) {
  async function wrapper({
    adapterId,
    fromChain,
    toChain,
    dryrun
  }: {
    adapterId: string
    fromChain: string
    toChain: string
    dryrun?: boolean
  }) {
    const fromChainId = await chainToId(db, fromChain)
    const toChainId = await chainToId(db, toChain)

    const query = `INSERT INTO Adapter (chainId, adapterId, data) SELECT ${toChainId}, adapterId, data FROM Adapter WHERE chainId=${fromChainId} and adapterId='${adapterId}'`

    if (dryrun) {
      console.debug(query)
    } else {
      const result = await db.run(query)
      console.log(formatResultInsert(result))
    }
  }
  return wrapper
}
