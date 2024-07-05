import { buildMockLogger } from '../src/logger'
import { REQUEST_RESPONSE_FULFILL_GAS_MINIMUM } from '../src/settings'
import { IRequestResponseListenerWorker } from '../src/types'
import { job } from '../src/worker/request-response'
import { JOB_ID_UINT128 } from './../src/worker/request-response.utils'
import { QUEUE } from './utils'

function KlayPriceRequest() {
  // "get": "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
  // "path": "RAW,KLAY,USD,PRICE"
  // "pow10": "8"
  return '0x63676574784a68747470733a2f2f6d696e2d6170692e63727970746f636f6d706172652e636f6d2f646174612f70726963656d756c746966756c6c3f6673796d733d4b4c4159267473796d733d5553446470617468725241572c4b4c41592c5553442c505249434565706f7731306138'
}

describe('Request-Response Worker', function () {
  it('Composability test', async function () {
    const logger = buildMockLogger()
    const wrapperFn = await job(QUEUE, logger)

    const callbackAddress = '0xccf9a654c878848991e46ab23d2ad055ca827979' // random address
    const sender = '0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266' // Hardhat Account #0

    const listenerData: IRequestResponseListenerWorker = {
      callbackAddress,
      blockNum: 1,
      requestId: '1',
      jobId: JOB_ID_UINT128,
      accId: '0',
      callbackGasLimit: 2500000,
      sender,
      isDirectPayment: false,
      numSubmission: 1,
      data: KlayPriceRequest(),
    }
    const tx = await wrapperFn({
      data: listenerData,
    })

    expect(tx?.gasLimit).toBe(REQUEST_RESPONSE_FULFILL_GAS_MINIMUM + listenerData.callbackGasLimit)
    expect(tx?.to).toBe(callbackAddress)
  })
})
