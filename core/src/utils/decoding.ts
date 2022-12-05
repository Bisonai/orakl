import { ethers } from 'ethers'
import { IRequest } from '../types.js'

const SIZELEN = 2

function hexToInt(hexNum, a) {
  const group = parseInt(hexNum, 16)

  if (group <= 119) {
    /* console.log('group1') */
    return [2, group - 97 + 1]
  } else if (group == 120) {
    /* console.log('group2') */
    return [4, parseInt(a.substring(0, 2), 16)]
  } else {
    console.log('we got problem')
    return [1, 1] // FIXME
  }
}

function hexToString(s) {
  return ethers.utils.toUtf8String(s)
}

function extractKeyOrValueLengths(obj) {
  const lengthHex = '0x' + obj.msg.substring(obj.counter, obj.counter + SIZELEN)
  const [sizeLength, keyOrValueLength] = hexToInt(
    lengthHex,
    obj.msg.substring(obj.counter + SIZELEN)
  )
  return [sizeLength, keyOrValueLength]
}

function extractKeyOrValue(obj, valueLength) {
  const keyOrValueHex = '0x' + obj.msg.substring(obj.counter, obj.counter + valueLength * SIZELEN)
  const keyOrValue = hexToString(keyOrValueHex)
  return keyOrValue
}

function readKeyOrValue(obj) {
  const [sizeLength, keyOrValueLength] = extractKeyOrValueLengths(obj)
  obj.counter += sizeLength

  const keyOrValue = extractKeyOrValue(obj, keyOrValueLength)
  obj.counter += keyOrValueLength * SIZELEN

  return keyOrValue
}

export function decodeAnyApiRequest(anyApiRequest: string): IRequest {
  if (anyApiRequest.substring(0, 2) == '0x') {
    anyApiRequest = anyApiRequest.substring(2)
  }

  let request = { get: '' }

  let obj = {
    msg: anyApiRequest,
    counter: 0
  }

  while (obj.counter < obj.msg.length) {
    const key = readKeyOrValue(obj)
    const value = readKeyOrValue(obj)

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
