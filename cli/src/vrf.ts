import { command, subcommands, option, string as cmdstring } from 'cmd-ts'
import ethers from 'ethers'
import { keygen } from '@bisonai/orakl-vrf'
import {
  dryrunOption,
  idOption,
  chainOptionalOption,
  chainToId,
  formatResultInsert,
  formatResultRemove
} from './utils'

export function vrfSub(db) {
  // vrf list   [--chain [chain]]                                                [--dryrun]
  // vrf insert  --chain [chain] --pk [pk] --sk [sk] --pk_x [pk_x] --pk_y [pk_y] [--dryrun]
  // vrf remove  --id [id]                                                       [--dryrun]
  // vrf keygen

  const list = command({
    name: 'list',
    args: {
      chain: chainOptionalOption,
      dryrun: dryrunOption
    },
    handler: listHandler(db, true)
  })

  const insert = command({
    name: 'insert',
    args: {
      chain: option({
        type: cmdstring,
        long: 'chain'
      }),
      sk: option({
        type: cmdstring,
        long: 'sk'
      }),
      pk: option({
        type: cmdstring,
        long: 'pk'
      }),
      pk_x: option({
        type: cmdstring,
        long: 'pk_x'
      }),
      pk_y: option({
        type: cmdstring,
        long: 'pk_y'
      }),
      key_hash: option({
        type: cmdstring,
        long: 'key_hash'
      }),
      dryrun: dryrunOption
    },
    handler: insertHandler(db)
  })

  const remove = command({
    name: 'remove',
    args: {
      id: idOption,
      dryrun: dryrunOption
    },
    handler: removeHandler(db)
  })

  const keygen = command({
    name: 'keygen',
    args: {},
    handler: keygenHandler(db)
  })

  return subcommands({
    name: 'vrf',
    cmds: { list, insert, remove, keygen }
  })
}

export function listHandler(db, print?: boolean) {
  async function wrapper({ chain, dryrun }: { chain?: string; dryrun?: boolean }) {
    let where = ''
    if (chain) {
      where += `AND Chain.name = '${chain}'`
    }
    const query = `SELECT VrfKey.id, Chain.name as chain, sk, pk, pk_x, pk_y, key_hash FROM VrfKey INNER JOIN Chain
   ON VrfKey.chainId = Chain.id ${where};`
    if (dryrun) {
      console.debug(query)
    } else {
      const result = await db.all(query)
      if (print) {
        console.log(result)
      }
      return result
    }
  }
  return wrapper
}

export function insertHandler(db) {
  async function wrapper({
    chain,
    pk,
    sk,
    pk_x,
    pk_y,
    key_hash,
    dryrun
  }: {
    chain: string
    pk: string
    sk: string
    pk_x: string
    pk_y: string
    key_hash: string
    dryrun?: boolean
  }) {
    const chainId = await chainToId(db, chain)
    const query = `INSERT INTO VrfKey (chainId, sk, pk, pk_x, pk_y, key_hash) VALUES (${chainId}, '${sk}', '${pk}', '${pk_x}', '${pk_y}', '${key_hash}');`
    if (dryrun) {
      console.debug(query)
    } else {
      const result = await db.run(query)
      console.log(formatResultInsert(result))
    }
  }
  return wrapper
}

export function removeHandler(db) {
  async function wrapper({ id, dryrun }: { id: number; dryrun?: boolean }) {
    const query = `DELETE FROM VrfKey WHERE id=${id};`
    if (dryrun) {
      console.debug(query)
    } else {
      const result = await db.run(query)
      console.log(formatResultRemove(result))
    }
  }
  return wrapper
}

export function keygenHandler(db) {
  async function wrapper() {
    const key = keygen()
    const pkX = key.public_key.x.toString()
    const pkY = key.public_key.y.toString()
    const keyHash = ethers.utils.solidityKeccak256(['uint256', 'uint256'], [pkX, pkY])

    console.log(`sk=${key.secret_key}`)
    console.log(`pk=${key.public_key.key}`)
    console.log(`pk_x=${pkX}`)
    console.log(`pk_y=${pkY}`)
    console.log(`key_hash=${keyHash}`)
  }
  return wrapper
}
