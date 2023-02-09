import { IRequest } from './types'
import { remove0x } from './utils'
import { IcnError, IcnErrorCode } from './errors'
import cbor from 'cbor'

export async function decodeRequest(anyApiRequest: string): Promise<IRequest> {
  anyApiRequest = remove0x(anyApiRequest)
  const buffer = Buffer.from(anyApiRequest, 'hex')
  const decodedMessage = await cbor.decodeAll(buffer)
  const request = { get: '' }

  // decodedMessage.length expected to be Even, pairs of Key and Value
  if (decodedMessage.length % 2 == 1) {
    throw new IcnError(IcnErrorCode.InvalidDecodedMesssageLength, decodedMessage.length.toString())
  }
  for (let i = 0; i < decodedMessage.length; i += 2) {
    const key = decodedMessage[i]
    const value = decodedMessage[i + 1]
    switch (key) {
      case 'get':
        request['get'] = value
        break
      case 'path':
        request['path'] = value.split(',')
        break
    }
  }
  return request
}
