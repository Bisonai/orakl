import { ethers } from 'ethers'
import { IAdapter, IAggregator } from '../types'

export async function computeDataHash({
  data,
  verify
}: {
  data: IAdapter | IAggregator
  verify?: boolean
}): Promise<IAdapter | IAggregator> {
  const input = JSON.parse(JSON.stringify(data))

  // Don't use `id` and `active` in hash computation
  delete input.id
  delete input.active

  const hash = ethers.utils.keccak256(ethers.utils.toUtf8Bytes(JSON.stringify(input)))

  if (verify && data.id != hash) {
    console.info(input)
    throw Error(`Hashes do not match!\nExpected ${hash}, received ${data.id}.`)
  } else {
    data.id = hash
    console.info(data)
    return data
  }
}
