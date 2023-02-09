import { describe, expect, test } from '@jest/globals'
import { decodeRequest } from '../src/worker/decoding'
import { add0x } from '../src/utils'
import cbor from 'cbor'

describe('Decode incoming request', function () {
  test('test getAndPath with CBOR', async function () {
    const request = {
      get: 'https://min-api.cryptocompare.com/data/pricemultifull?fsyms=ETH&tsyms=USD',
      path: 'RAW,ETH,USD,PRICE'
    }

    let bufferList: Buffer = Buffer.from('')
    for (const key in request) {
      bufferList = Buffer.concat([bufferList, cbor.encode(key), cbor.encode(request[key])])
    }
    const hexValue = add0x(bufferList.toString('hex'))
    const decodedRequest = await decodeRequest(hexValue)

    expect(decodedRequest[0].input).toStrictEqual(request.get)
    expect(decodedRequest[1].input).toStrictEqual(request.path)
  })
})
