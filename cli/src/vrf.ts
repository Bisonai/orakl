import { keygen } from '@bisonai/orakl-vrf'
import axios from 'axios'
import { command, option, string as cmdstring, subcommands } from 'cmd-ts'
import ethers from 'ethers'
import { ORAKL_NETWORK_API_URL } from './settings.js'
import { buildUrl, chainOptionalOption, idOption, isOraklNetworkApiHealthy } from './utils.js'

const VRF_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'vrf')

export function vrfSub() {
  // vrf list   [--chain ${chain}]
  // vrf insert  --chain ${chain} --pk ${pk} --sk ${sk} --pkX ${pk_x} --pkY ${pkY} --keyHash ${keyHash}
  // vrf remove  --id ${id}
  // vrf keygen

  const list = command({
    name: 'list',
    args: {
      chain: chainOptionalOption,
    },
    handler: listHandler(true),
  })

  const insert = command({
    name: 'insert',
    args: {
      chain: option({
        type: cmdstring,
        long: 'chain',
      }),
      sk: option({
        type: cmdstring,
        long: 'sk',
      }),
      pk: option({
        type: cmdstring,
        long: 'pk',
      }),
      pkX: option({
        type: cmdstring,
        long: 'pkX',
      }),
      pkY: option({
        type: cmdstring,
        long: 'pkY',
      }),
      keyHash: option({
        type: cmdstring,
        long: 'keyHash',
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

  const keygen = command({
    name: 'keygen',
    args: {},
    handler: keygenHandler(),
  })

  return subcommands({
    name: 'vrf',
    cmds: { list, insert, remove, keygen },
  })
}

export function listHandler(print?: boolean) {
  async function wrapper({ chain }: { chain?: string }) {
    if (!(await isOraklNetworkApiHealthy())) return

    try {
      if (!chain) {
        const result = (await axios.get(VRF_ENDPOINT))?.data
        if (print) {
          console.dir(result, { depth: null })
        }
        return result
      }
      const result = (await axios.get(VRF_ENDPOINT, { data: { chain } }))?.data
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
    pk,
    sk,
    pkX,
    pkY,
    keyHash,
  }: {
    chain: string
    pk: string
    sk: string
    pkX: string
    pkY: string
    keyHash: string
  }) {
    if (!(await isOraklNetworkApiHealthy())) return

    try {
      const result = (await axios.post(VRF_ENDPOINT, { chain, pk, sk, pkX, pkY, keyHash })).data
      console.dir(result, { depth: null })
      return result
    } catch (e) {
      console.error('VRF key was not inserted. Reason:')
      console.error(e?.response?.data?.message)
      return e?.response?.data?.message
    }
  }
  return wrapper
}

export function removeHandler() {
  async function wrapper({ id }: { id: number }) {
    if (!(await isOraklNetworkApiHealthy())) return

    try {
      const endpoint = buildUrl(VRF_ENDPOINT, id.toString())
      const result = (await axios.delete(endpoint)).data
      console.dir(result, { depth: null })
    } catch (e) {
      console.error('VRF key was not deleted. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function keygenHandler() {
  async function wrapper() {
    const key = keygen()
    const pkX = key.public_key.x.toString()
    const pkY = key.public_key.y.toString()
    const keyHash = ethers.utils.solidityKeccak256(['uint256', 'uint256'], [pkX, pkY])

    console.log(`sk=${key.secret_key}`)
    console.log(`pk=${key.public_key.key}`)
    console.log(`pkX=${pkX}`)
    console.log(`pkY=${pkY}`)
    console.log(`keyHash=${keyHash}`)
  }
  return wrapper
}
