import { describe, expect, test } from '@jest/globals'
import {
  contractConnectHandler,
  contractInsertHandler,
  contractRemoveHandler,
  functionInsertHandler,
  functionRemoveHandler,
  organizationInsertHandler,
  organizationRemoveHandler,
  reporterInsertHandler,
  reporterRemoveHandler,
} from '../src/delegator'

// const unsignedTx = {
//   from: '0x260836ac4f046b6887bbe16b322e7f1e5f9a0452',
//   to: '0x93120927379723583c7a0dd2236fcb255e96949f',
//   input: '0xd09de08a',
//   gas: '0x15f90',
//   value: '0x0',
//   chainId: '0x3e9',
//   gasPrice: '0xba43b7400',
//   nonce: '0x84',
//   v: '0x07f5',
//   r: '0x3cfdcaac218e807ad61ef0064c87fcd5604b99263ee37a082bbb9e497121b6a2',
//   s: '0x303ddd61ff92f2c2513b67729d7a92cede3233f16016b15b9c25ef4c35492075',
//   rawTx:
//     '0x31f89f8184850ba43b740083015f909493120927379723583c7a0dd2236fcb255e96949f8094260836ac4f046b6887bbe16b322e7f1e5f9a045284d09de08af847f8458207f5a03cfdcaac218e807ad61ef0064c87fcd5604b99263ee37a082bbb9e497121b6a2a0303ddd61ff92f2c2513b67729d7a92cede3233f16016b15b9c25ef4c35492075940000000000000000000000000000000000000000c4c3018080'
// }
const organizationName = 'BISONAI'
const reporterAddress = '0x260836ac4f046b6887bbe16b322e7f1e5f9a0452'
const contractAddress = '0x93120927379723583c7a0dd2236fcb255e96949f'
const functionName = 'increment()'

describe('CLI Delegator', function () {
  test('Test Delegator', async function () {
    // Insert Organization
    const organization = await organizationInsertHandler()({ name: organizationName })
    expect(organization.name).toBe(organizationName)

    //Insert Reporter
    const reporter = await reporterInsertHandler()({
      address: reporterAddress,
      organizationId: Number(organization.id),
    })
    expect(reporter.address).toBe(reporterAddress)
    expect(Number(reporter.organizationId)).toBe(Number(organization.id))

    // Insert Contract
    const contract = await contractInsertHandler()({ address: contractAddress })
    expect(contract.address).toBe(contractAddress)

    // Insert Functions
    const functions = await functionInsertHandler()({
      name: functionName,
      contractId: Number(contract.id),
    })
    expect(functions.name).toBe(functionName)

    // Connect contract with reporter
    await contractConnectHandler()({
      contractId: Number(contract.id),
      reporterId: Number(reporter.id),
    })

    // Sign Transaction
    // const signedTx = await signHandler()({ txData: unsignedTx })
    // expect(signedTx.succeed).toBe(true)
    // console.log(signedTx)

    await functionRemoveHandler()({ id: functions.id })
    await contractRemoveHandler()({ id: contract.id })
    await reporterRemoveHandler()({ id: reporter.id })
    await organizationRemoveHandler()({ id: organization.id })
  })
})
