import { describe, expect, test } from '@jest/globals'
import { postprocessListeners } from '../src/settings'

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
})
