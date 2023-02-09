import { describe, expect, test } from '@jest/globals'
import { decodeRequest } from '../src/decoding'
import cbor from 'cbor'

describe('Decode incoming request', function () {
  test('test getAndPath with CBOR', async function () {
    const anyApi = {
      get: 'https://min-api.cryptocompare.com/data/pricemultifull?fsyms=ETH&tsyms=USD',
      path: 'RAW,ETH,USD,PRICE'
    }

    let bufferList: Buffer = Buffer.from('')
    for (const key in anyApi) {
      bufferList = Buffer.concat([bufferList, cbor.encode(key), cbor.encode(anyApi[key])])
    }
    const hexValue = '0x' + bufferList.toString('hex')
    const request = await decodeRequest(hexValue)

    expect(request.get).toStrictEqual(anyApi.get)
    expect(request.path).toStrictEqual(anyApi.path.split(','))
  })
})
