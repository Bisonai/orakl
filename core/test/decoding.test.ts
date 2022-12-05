import { describe, expect, test } from '@jest/globals'
import { decodeAnyApiRequest } from '../src/decoding'

describe('Decode incoming request', function () {
  test('getAndPath', async function () {
    const anyApiRequest =
      '0x63676574784968747470733a2f2f6d696e2d6170692e63727970746f636f6d706172652e636f6d2f646174612f70726963656d756c746966756c6c3f6673796d733d455448267473796d733d5553446470617468715241572c4554482c5553442c5052494345'
    const request = decodeAnyApiRequest(anyApiRequest)
    expect(request.get).toStrictEqual(
      'https://min-api.cryptocompare.com/data/pricemultifull?fsyms=ETH&tsyms=USD'
    )
    expect(request.path).toStrictEqual(['RAW', 'ETH', 'USD', 'PRICE'])
  })
})
