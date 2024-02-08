import { describe } from '@jest/globals'
import {
  listHandler as adapterListHandler,
  removeHandler as adapterRemoveHandler
} from '../src/adapter'
import {
  listHandler as aggregatorListHandler,
  removeHandler as aggregatorRemoveHandler
} from '../src/aggregator'
import {
  insertHandler as chainInsertHandler,
  listHandler as chainListHandler,
  removeHandler as chainRemoveHandler
} from '../src/chain'
import { bulkInsertHandler, bulkRemoveHandler } from '../src/datafeed'
import {
  contractListHandler,
  contractRemoveHandler,
  functionListHandler,
  functionRemoveHandler,
  organizationInsertHandler,
  organizationListHandler,
  organizationRemoveHandler,
  reporterListHandler as delegatorReporterListHandler,
  reporterRemoveHandler as delegatorReporterRemoveHandler
} from '../src/delegator'
import {
  listHandler as listenerListHandler,
  removeHandler as listenerRemoveHandler
} from '../src/listener'
import {
  listHandler as reporterListHandler,
  removeHandler as reporterRemoveHandler
} from '../src/reporter'
import {
  insertHandler as serviceInsertHandler,
  listHandler as serviceListHandler,
  removeHandler as serviceRemoveHandler
} from '../src/service'
import { DATAFEED_BULK_0, DATAFEED_BULK_1 } from './mockData'

describe('CLI datafeed', function () {
  beforeAll(async () => {
    await chainInsertHandler()({ name: 'localhost' })
    await chainInsertHandler()({ name: 'baobab' })
    await serviceInsertHandler()({ name: 'DATA_FEED' })
    await serviceInsertHandler()({ name: 'DATA_FEED_V2' })
    await organizationInsertHandler()({ name: 'bisonai' })
    await organizationInsertHandler()({ name: 'kf' })
  })

  afterAll(async () => {
    const chains = await chainListHandler()()
    for (const chain of chains) {
      await chainRemoveHandler()({ id: chain.id })
    }
    const services = await serviceListHandler()()
    for (const service of services) {
      await serviceRemoveHandler()({ id: service.id })
    }
    const organizations = await organizationListHandler()()
    for (const organization of organizations) {
      await organizationRemoveHandler()({ id: organization.id })
    }
  })

  afterEach(async () => {
    const afterAdapterList = await adapterListHandler()()
    const afterAggregatorList = await aggregatorListHandler()({})
    const afterDelegatorReporterList = await delegatorReporterListHandler()()
    const afterContractList = await contractListHandler()()
    const afterListenerList = await listenerListHandler()({})
    const afterReporterList = await reporterListHandler()({})
    const afterFunctionList = await functionListHandler()()

    for (const reporter of afterReporterList) {
      await reporterRemoveHandler()({ id: reporter.id })
    }
    for (const listener of afterListenerList) {
      await listenerRemoveHandler()({ id: listener.id })
    }
    for (const _function of afterFunctionList) {
      await functionRemoveHandler()({ id: Number(_function.id) })
    }
    for (const aggregator of afterAggregatorList) {
      await aggregatorRemoveHandler()({ id: aggregator.id })
    }
    for (const adapter of afterAdapterList) {
      await adapterRemoveHandler()({ id: adapter.id })
    }
    for (const delegatorReporter of afterDelegatorReporterList) {
      await delegatorReporterRemoveHandler()({ id: delegatorReporter.id })
    }
    for (const contract of afterContractList) {
      await contractRemoveHandler()({ id: contract.id })
    }
  })

  test('datafeed bulk insert with default values', async function () {
    const beforeAdapterList = await adapterListHandler()()
    const beforeAggregatorList = await aggregatorListHandler()({})
    const beforeDelegatorReporterList = await delegatorReporterListHandler()()
    const beforeContractList = await contractListHandler()()
    const beforeListenerList = await listenerListHandler()({})
    const beforeReporterList = await reporterListHandler()({})
    const beforeFunctionList = await functionListHandler()()

    await bulkInsertHandler()({ data: DATAFEED_BULK_0 })

    const bulkLength = DATAFEED_BULK_0.bulk.length

    const afterAdapterList = await adapterListHandler()()
    const afterAggregatorList = await aggregatorListHandler()({})
    const afterDelegatorReporterList = await delegatorReporterListHandler()()
    const afterContractList = await contractListHandler()()
    const afterListenerList = await listenerListHandler()({})
    const afterReporterList = await reporterListHandler()({})
    const afterFunctionList = await functionListHandler()()

    expect(afterAdapterList.length).toEqual(beforeAdapterList.length + bulkLength)
    expect(afterAggregatorList.length).toEqual(beforeAggregatorList.length + bulkLength)
    expect(afterDelegatorReporterList.length).toEqual(
      beforeDelegatorReporterList.length + bulkLength
    )
    expect(afterContractList.length).toEqual(beforeContractList.length + bulkLength)
    expect(afterListenerList.length).toEqual(beforeListenerList.length + bulkLength)
    expect(afterReporterList.length).toEqual(beforeReporterList.length + bulkLength)
    expect(afterFunctionList.length).toEqual(beforeFunctionList.length + bulkLength)
  })

  test('datafeed bulk insert', async function () {
    const beforeAdapterList = await adapterListHandler()()
    const beforeAggregatorList = await aggregatorListHandler()({})
    const beforeDelegatorReporterList = await delegatorReporterListHandler()()
    const beforeContractList = await contractListHandler()()
    const beforeListenerList = await listenerListHandler()({})
    const beforeReporterList = await reporterListHandler()({})
    const beforeFunctionList = await functionListHandler()()

    await bulkInsertHandler()({ data: DATAFEED_BULK_1 })

    const bulkLength = DATAFEED_BULK_1.bulk.length

    const afterAdapterList = await adapterListHandler()()
    const afterAggregatorList = await aggregatorListHandler()({})
    const afterDelegatorReporterList = await delegatorReporterListHandler()()
    const afterContractList = await contractListHandler()()
    const afterListenerList = await listenerListHandler()({})
    const afterReporterList = await reporterListHandler()({})
    const afterFunctionList = await functionListHandler()()

    expect(afterAdapterList.length).toEqual(beforeAdapterList.length + bulkLength)
    expect(afterAggregatorList.length).toEqual(beforeAggregatorList.length + bulkLength)
    expect(afterDelegatorReporterList.length).toEqual(
      beforeDelegatorReporterList.length + bulkLength
    )
    expect(afterContractList.length).toEqual(beforeContractList.length + bulkLength)
    expect(afterListenerList.length).toEqual(beforeListenerList.length + bulkLength)
    expect(afterReporterList.length).toEqual(beforeReporterList.length + bulkLength)
    expect(afterFunctionList.length).toEqual(beforeFunctionList.length + bulkLength)
  })

  test('datafeed bulk removal', async function () {
    const beforeAdapterList = await adapterListHandler()()
    const beforeAggregatorList = await aggregatorListHandler()({})
    const beforeDelegatorReporterList = await delegatorReporterListHandler()()
    const beforeContractList = await contractListHandler()()
    const beforeListenerList = await listenerListHandler()({})
    const beforeReporterList = await reporterListHandler()({})
    const beforeFunctionList = await functionListHandler()()

    await bulkInsertHandler()({ data: DATAFEED_BULK_0 })

    const bulkLength = DATAFEED_BULK_0.bulk.length

    const afterAdapterList = await adapterListHandler()()
    const afterAggregatorList = await aggregatorListHandler()({})
    const afterDelegatorReporterList = await delegatorReporterListHandler()()
    const afterContractList = await contractListHandler()()
    const afterListenerList = await listenerListHandler()({})
    const afterReporterList = await reporterListHandler()({})
    const afterFunctionList = await functionListHandler()()

    expect(afterAdapterList.length).toEqual(beforeAdapterList.length + bulkLength)
    expect(afterAggregatorList.length).toEqual(beforeAggregatorList.length + bulkLength)
    expect(afterDelegatorReporterList.length).toEqual(
      beforeDelegatorReporterList.length + bulkLength
    )
    expect(afterContractList.length).toEqual(beforeContractList.length + bulkLength)
    expect(afterListenerList.length).toEqual(beforeListenerList.length + bulkLength)
    expect(afterReporterList.length).toEqual(beforeReporterList.length + bulkLength)
    expect(afterFunctionList.length).toEqual(beforeFunctionList.length + bulkLength)

    await bulkRemoveHandler()({ data: DATAFEED_BULK_0 })

    const afterDeleteDelegatorReporterList = await delegatorReporterListHandler()()
    const afterDeleteContractList = await contractListHandler()()
    const afterDeleteListenerList = await listenerListHandler()({})
    const afterDeleteReporterList = await reporterListHandler()({})
    const afterDeleteFunctionList = await functionListHandler()()

    expect(afterDeleteDelegatorReporterList.length).toEqual(0)
    expect(afterDeleteContractList.length).toEqual(0)
    expect(afterDeleteListenerList.length).toEqual(0)
    expect(afterDeleteReporterList.length).toEqual(0)
    expect(afterDeleteFunctionList.length).toEqual(0)
  })
})
