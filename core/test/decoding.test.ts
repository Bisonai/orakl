import { describe, expect, test } from '@jest/globals'
import cbor from 'cbor'
import { add0x } from '../src/utils'
import { decodeRequest } from '../src/worker/decoding'

describe('Decode incoming request', function () {
  test('test getAndPath with CBOR', async function () {
    const request = {
      get: 'https://min-api.cryptocompare.com/data/pricemultifull?fsyms=ETH&tsyms=USD',
      path: 'RAW,ETH,USD,PRICE',
    }

    const b: Buffer[] = []
    for (const key in request) {
      b.push(cbor.encode(key))
      b.push(cbor.encode(request[key]))
    }

    const buffer = Buffer.concat(b).toString('hex')
    const decodedRequest = await decodeRequest(add0x(buffer))

    expect(decodedRequest[0].args).toStrictEqual(request.get)
    expect(decodedRequest[1].args).toStrictEqual(request.path)
  })
})
