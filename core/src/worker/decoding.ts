import { IRequestOperation } from '../types'
import { remove0x } from '../utils'
import { IcnError, IcnErrorCode } from '../errors'
import cbor from 'cbor'

export async function decodeRequest(anyApiRequest: string): Promise<IRequestOperation[]> {
  anyApiRequest = remove0x(anyApiRequest)
  const buffer = Buffer.from(anyApiRequest, 'hex')
  const decodedMessage = await cbor.decodeAll(buffer)
  const request: IRequestOperation[] = []

  // decodedMessage.length is expected to be even, pairs of Key and Value
  if (decodedMessage.length % 2 == 1) {
    throw new IcnError(IcnErrorCode.InvalidDecodedMesssageLength, decodedMessage.length.toString())
  }

  for (let i = 0; i < decodedMessage.length; i += 2) {
    request.push({ opcode: decodedMessage[i], input: decodedMessage[i + 1] })
  }

  return request
}
