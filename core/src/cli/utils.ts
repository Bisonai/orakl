import { ethers } from 'ethers'
import { Logger } from 'pino'
import { IAdapter, IAggregator } from '../types'

export async function computeDataHash({
  data,
  verify,
  logger
}: {
  data: IAdapter | IAggregator
  verify?: boolean
  logger?: Logger
}): Promise<IAdapter | IAggregator> {
  const input = JSON.parse(JSON.stringify(data))

  // Don't use `id` and `active` in hash computation
  delete input.id
  delete input.active

  const hash = ethers.utils.keccak256(ethers.utils.toUtf8Bytes(JSON.stringify(input)))

  if (verify && data.id != hash) {
    logger?.info(input)
    throw Error(`Hashes do not match!\nExpected ${hash}, received ${data.id}.`)
  } else {
    data.id = hash
    logger?.info(data)
    return data
  }
}
