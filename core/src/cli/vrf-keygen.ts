import ethers from 'ethers'
import { keygen } from '../vrf/index'

async function main() {
  const key = keygen()

  console.log(`SK=${key.secret_key}`)
  console.log(`PK=${key.public_key.key}`)

  const VRF_PK_X = key.public_key.x.toString()
  const VRF_PK_Y = key.public_key.y.toString()
  console.log(`PK_X=${VRF_PK_X}`)
  console.log(`PK_Y=${VRF_PK_Y}`)

  const KEY_HASH = ethers.utils.solidityKeccak256(['uint256', 'uint256'], [VRF_PK_X, VRF_PK_Y])
  console.log(`KEY_HASH=${KEY_HASH}`)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
