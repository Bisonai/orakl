import * as Path from 'node:path'
import { loadJson } from '../utils.js'
import { ethers } from 'ethers'
import { ADAPTER_ROOT_DIR } from '../settings.js'

import { parseArgs } from 'node:util'

async function getAdapterHash(fileName) {
  const data = await loadJson(Path.join(ADAPTER_ROOT_DIR, fileName))
  const hash = ethers.utils.keccak256(ethers.utils.toUtf8Bytes(JSON.stringify(data.feeds)))
  console.log('Adapter Hash:', hash)
  return hash
}

async function main() {
  const values = loadArgs()
  console.log(values)
}

function loadArgs() {
  const {
    values: { input, output, verify }
  } = parseArgs({
    options: {
      input: {
        type: 'string'
      },
      output: {
        type: 'string'
      },
      verify: {
        type: 'boolean'
      }
    }
  })

  return { input, output, verify }
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
