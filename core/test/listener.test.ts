import { describe, expect, test } from '@jest/globals'
import { groupListeners, validateListenerConfig } from '../src/listener/utils'

describe('Listener', function () {
  test('Postprocess listeners', function () {
    const input = [
      {
        service: 'Service1',
        address: '0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0',
        eventName: 'RandomWordsRequested',
      },
      {
        service: 'Service2',
        address: '0xa513E6E4b8f2a923D98304ec87F64353C4D5C853',
        eventName: 'NewRound',
      },
      {
        service: 'Service1',
        address: '0x45778c29A34bA00427620b937733490363839d8C',
        eventName: 'Requested',
      },
    ]

    const expectedOutput = {
      Service1: [
        {
          address: '0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0',
          eventName: 'RandomWordsRequested',
        },
        {
          address: '0x45778c29A34bA00427620b937733490363839d8C',
          eventName: 'Requested',
        },
      ],
      Service2: [
        {
          address: '0xa513E6E4b8f2a923D98304ec87F64353C4D5C853',
          eventName: 'NewRound',
        },
      ],
    }
    const output = groupListeners({ listenersRawConfig: input })
    expect(expectedOutput).toStrictEqual(output)
  })

  test('Should pass the validation of listener config', function () {
    const config = [
      {
        id: '0',
        address: '0x0165878A594ca255338adfa4d48449f69242Eb8F',
        eventName: 'RandomWordsRequested',
        chain: 'localhost',
      },
    ]
    const isValid = validateListenerConfig(config)
    expect(isValid).toBe(true)
  })

  test('Should fail the validation of listener config', function () {
    const config = [
      {
        id: '0',
        address: '0x0165878A594ca255338adfa4d48449f69242Eb8F',
        //eventName: 'RandomWordsRequested'
        chain: 'localhost',
      },
    ]
    const isValid = validateListenerConfig(config as any) // eslint-disable-line @typescript-eslint/no-explicit-any
    expect(isValid).toBe(false)
  })
})
