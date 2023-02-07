import { describe, expect, test } from '@jest/globals'
import { postprocessListeners } from '../src/settings'
import { validateListenerConfig } from '../src/listener/utils'

describe('Listener', function () {
  test('Postprocess listeners', function () {
    const input = [
      {
        name: 'Service1',
        address: '0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0',
        eventName: 'RandomWordsRequested'
      },
      {
        name: 'Service2',
        address: '0xa513E6E4b8f2a923D98304ec87F64353C4D5C853',
        eventName: 'NewRound'
      },
      {
        name: 'Service1',
        address: '0x45778c29A34bA00427620b937733490363839d8C',
        eventName: 'Requested'
      }
    ]

    const expectedOutput = {
      Service1: [
        {
          address: '0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0',
          eventName: 'RandomWordsRequested'
        },
        {
          address: '0x45778c29A34bA00427620b937733490363839d8C',
          eventName: 'Requested'
        }
      ],
      Service2: [
        {
          address: '0xa513E6E4b8f2a923D98304ec87F64353C4D5C853',
          eventName: 'NewRound'
        }
      ]
    }
    const output = postprocessListeners(input)
    expect(expectedOutput).toStrictEqual(output)
  })

  test('Should pass the validation of listener config', function () {
    const config = [
      {
        address: '0x0165878a594ca255338adfa4d48449f69242eb8f',
        eventName: 'RandomWordsRequested'
      }
    ]
    const isValid = validateListenerConfig(config)
    expect(isValid).toBe(true)
  })

  test('Should fail the validation of listener config', function () {
    const config = [
      {
        address: '0x0165878a594ca255338adfa4d48449f69242eb8f'
        //eventName: 'RandomWordsRequested'
      }
    ]
    const isValid = validateListenerConfig(config as any)
    expect(isValid).toBe(false)
  })
})
