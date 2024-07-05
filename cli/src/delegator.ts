import axios from 'axios'
import { command, number as cmdnumber, option, string as cmdstring, subcommands } from 'cmd-ts'
import { ORAKL_NETWORK_API_URL, ORAKL_NETWORK_DELEGATOR_URL } from './settings.js'
import { buildUrl, idOption, isOraklDelegatorHealthy } from './utils.js'

const AGGREGATOR_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'aggregator')

export function delegatorSub() {
  // delegator sign
  // delegator signList

  // delegator organizationList
  // delegator organizationInsert --name {organizationName}
  // delegator organizationRemove --id {organizationId}

  // delegator reporterList
  // delegator reporterInsert --address {reporterAddress} --organizationId {organizationId}
  // delegator reporterRemove --id {reporterId}

  // delegator contractList
  // delegator contractInsert --address {contractAddress}
  // delegator contractRemove --id {reporterId}
  // delegator contractConnect --contractId {contractId} -- repoterId {reporterId}
  // delegator contractDisconnect --contractId {contractId} -- repoterId {reporterId}

  // delegator functionList
  // delegator functionInsert --address {contractAddress} --organizationId {organizationId}
  // delegator functionRemove --id {reporterId}

  const sign = command({
    name: 'sign',
    args: {
      txData: option({
        type: cmdstring,
        long: 'txData',
      }),
    },
    handler: signHandler(),
  })

  const signList = command({
    name: 'sign',
    args: {},
    handler: signListHandler(),
  })
  const organizationList = command({
    name: 'organizationList',
    args: {},
    handler: organizationListHandler(),
  })

  const organizationInsert = command({
    name: 'organizationInsert',
    args: {
      name: option({
        type: cmdstring,
        long: 'name',
      }),
    },
    handler: organizationInsertHandler(),
  })

  const organizationRemove = command({
    name: 'organizationRemove',
    args: {
      id: idOption,
    },
    handler: organizationRemoveHandler(),
  })

  const reporterList = command({
    name: 'reporterList',
    args: {},
    handler: reporterListHandler(),
  })

  const reporterInsert = command({
    name: 'reporterInsert',
    args: {
      address: option({
        type: cmdstring,
        long: 'address',
      }),
      organizationId: option({
        type: cmdnumber,
        long: 'organizationId',
      }),
    },
    handler: reporterInsertHandler(),
  })

  const reporterRemove = command({
    name: 'reporterRemove',
    args: {
      id: idOption,
    },
    handler: reporterRemoveHandler(),
  })

  const contractList = command({
    name: 'contractList',
    args: {},
    handler: contractListHandler(),
  })

  const contractInsert = command({
    name: 'contractInsert',
    args: {
      address: option({
        type: cmdstring,
        long: 'address',
      }),
    },
    handler: contractInsertHandler(),
  })

  const contractRemove = command({
    name: 'contractRemove',
    args: {
      id: idOption,
    },
    handler: contractRemoveHandler(),
  })

  const contractConnect = command({
    name: 'contractConnect',
    args: {
      contractId: option({
        type: cmdnumber,
        long: 'contractId',
      }),
      reporterId: option({
        type: cmdnumber,
        long: 'reporterId',
      }),
    },
    handler: contractConnectHandler(),
  })

  const contractDisconnect = command({
    name: 'contractConnect',
    args: {
      contractId: option({
        type: cmdnumber,
        long: 'contractId',
      }),
      reporterId: option({
        type: cmdnumber,
        long: 'reporterId',
      }),
    },
    handler: contractDisconnectHandler(),
  })

  const functionList = command({
    name: 'functionList',
    args: {},
    handler: functionListHandler(),
  })

  const functionInsert = command({
    name: 'functionInsert',
    args: {
      name: option({
        type: cmdstring,
        long: 'name',
      }),
      contractId: option({
        type: cmdnumber,
        long: 'contractId',
      }),
    },
    handler: functionInsertHandler(),
  })

  const functionRemove = command({
    name: 'functionRemove',
    args: {
      id: idOption,
    },
    handler: functionRemoveHandler(),
  })

  return subcommands({
    name: 'delegator',
    cmds: {
      sign,
      signList,
      organizationList,
      organizationInsert,
      organizationRemove,
      reporterList,
      reporterInsert,
      reporterRemove,
      contractList,
      contractInsert,
      contractRemove,
      contractDisconnect,
      contractConnect,
      functionList,
      functionInsert,
      functionRemove,
    },
  })
}

export function signHandler() {
  async function wrapper({ txData }: { txData }) {
    if (!(await isOraklDelegatorHealthy())) return
    try {
      const endpoint = buildUrl(ORAKL_NETWORK_DELEGATOR_URL, `sign`)
      const result = (await axios.post(endpoint, { ...txData })).data
      console.log(result)
      return result
    } catch (e) {
      console.error('Delegator sign was not Signed. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function signListHandler() {
  async function wrapper() {
    if (!(await isOraklDelegatorHealthy())) return

    try {
      const endpoint = buildUrl(ORAKL_NETWORK_DELEGATOR_URL, `sign`)
      const result = (await axios.get(endpoint)).data
      console.log(result)
      return result
    } catch (e) {
      console.error('Delegator Sign was not listed. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function organizationListHandler() {
  async function wrapper() {
    if (!(await isOraklDelegatorHealthy())) return

    try {
      const endpoint = buildUrl(ORAKL_NETWORK_DELEGATOR_URL, `organization`)
      const result = (await axios.get(endpoint)).data
      console.log(result)
      return result
    } catch (e) {
      console.error('Delegator Organization was not listed. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function organizationInsertHandler() {
  async function wrapper({ name }: { name: string }) {
    if (!(await isOraklDelegatorHealthy())) return

    try {
      const endpoint = buildUrl(ORAKL_NETWORK_DELEGATOR_URL, `organization`)
      const result = (await axios.post(endpoint, { name })).data
      console.log(result)
      return result
    } catch (e) {
      console.error('Delegator Organization was not inserted. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function organizationRemoveHandler() {
  async function wrapper({ id }: { id: number }) {
    if (!(await isOraklDelegatorHealthy())) return

    try {
      const endpoint = buildUrl(
        buildUrl(ORAKL_NETWORK_DELEGATOR_URL, `organization`),
        id.toString(),
      )
      const result = (await axios.delete(endpoint)).data
      console.log(result)
      return result
    } catch (e) {
      console.error('Delegator Organization was not deleted. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function reporterListHandler() {
  async function wrapper() {
    if (!(await isOraklDelegatorHealthy())) return

    try {
      const endpoint = buildUrl(ORAKL_NETWORK_DELEGATOR_URL, `reporter`)
      const result = (await axios.get(endpoint)).data

      const printResult: any[] = []
      const aggregatorUrl = new URL(AGGREGATOR_ENDPOINT)
      const aggregatorResult = (await axios.get(aggregatorUrl.toString())).data

      for (const reporter of result) {
        if (!reporter.contract) {
          printResult.push({ ...reporter })
          continue
        }

        const aggregator = aggregatorResult.find(
          (aggregator) => aggregator.address === reporter.contract[0],
        )
        if (aggregator) {
          printResult.push({ ...reporter, name: aggregator.name })
        } else {
          printResult.push({ ...reporter })
        }
      }

      console.log(printResult)
      return result
    } catch (e) {
      console.error('Delegator Reporter was not listed. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function reporterInsertHandler() {
  async function wrapper({ address, organizationId }: { address: string; organizationId: number }) {
    if (!(await isOraklDelegatorHealthy())) return

    try {
      const endpoint = buildUrl(ORAKL_NETWORK_DELEGATOR_URL, `reporter`)
      const result = (await axios.post(endpoint, { address, organizationId })).data
      console.log(result)
      return result
    } catch (e) {
      console.error('Delegator Reporter was not inserted. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function reporterRemoveHandler() {
  async function wrapper({ id }: { id: number }) {
    if (!(await isOraklDelegatorHealthy())) return

    try {
      const endpoint = buildUrl(buildUrl(ORAKL_NETWORK_DELEGATOR_URL, `reporter`), id.toString())
      const result = (await axios.delete(endpoint)).data
      console.log(result)
      return result
    } catch (e) {
      console.error('Delegator Reporter was not deleted. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function contractListHandler() {
  async function wrapper() {
    if (!(await isOraklDelegatorHealthy())) return

    try {
      const endpoint = buildUrl(ORAKL_NETWORK_DELEGATOR_URL, `contract`)
      const result = (await axios.get(endpoint)).data
      console.log(result)
      return result
    } catch (e) {
      console.error('Delegator Contract was not listed. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function contractInsertHandler() {
  async function wrapper({ address }: { address: string }) {
    if (!(await isOraklDelegatorHealthy())) return

    try {
      const endpoint = buildUrl(ORAKL_NETWORK_DELEGATOR_URL, `contract`)
      const result = (await axios.post(endpoint, { address })).data
      console.log(result)
      return result
    } catch (e) {
      console.error('Delegator Contract was not inserted. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function contractRemoveHandler() {
  async function wrapper({ id }: { id: number }) {
    if (!(await isOraklDelegatorHealthy())) return

    try {
      const endpoint = buildUrl(buildUrl(ORAKL_NETWORK_DELEGATOR_URL, `contract`), id.toString())
      const result = (await axios.delete(endpoint)).data
      console.log(result)
      return result
    } catch (e) {
      console.error('Delegator Contract was not deleted. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function contractConnectHandler() {
  async function wrapper({ contractId, reporterId }: { contractId: number; reporterId: number }) {
    if (!(await isOraklDelegatorHealthy())) return

    try {
      const endpoint = buildUrl(ORAKL_NETWORK_DELEGATOR_URL, `contract/connectReporter`)
      const result = (await axios.post(endpoint, { contractId, reporterId })).data
      console.log(result)
      return result
    } catch (e) {
      console.error('Delegator Organization was not listed. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function contractDisconnectHandler() {
  async function wrapper({ contractId, reporterId }: { contractId: number; reporterId: number }) {
    if (!(await isOraklDelegatorHealthy())) return

    try {
      const endpoint = buildUrl(ORAKL_NETWORK_DELEGATOR_URL, `contract/disconnectReporter`)
      const result = (await axios.post(endpoint, { contractId, reporterId })).data
      console.log(result)
      return result
    } catch (e) {
      console.error('Delegator Organization was not listed. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function functionListHandler() {
  async function wrapper() {
    if (!(await isOraklDelegatorHealthy())) return

    try {
      const endpoint = buildUrl(ORAKL_NETWORK_DELEGATOR_URL, `function`)
      const result = (await axios.get(endpoint)).data
      console.log(result)
      return result
    } catch (e) {
      console.error('Delegator Function was not listed. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function functionInsertHandler() {
  async function wrapper({ name, contractId }: { name: string; contractId: number }) {
    if (!(await isOraklDelegatorHealthy())) return

    try {
      const endpoint = buildUrl(ORAKL_NETWORK_DELEGATOR_URL, `function`)
      const result = (await axios.post(endpoint, { name, contractId })).data
      console.log(result)
      return result
    } catch (e) {
      console.error('Delegator Function was not inserted. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function functionRemoveHandler() {
  async function wrapper({ id }: { id: number }) {
    if (!(await isOraklDelegatorHealthy())) return

    try {
      const endpoint = buildUrl(buildUrl(ORAKL_NETWORK_DELEGATOR_URL, `function`), id.toString())
      const result = (await axios.delete(endpoint)).data
      console.log(result)
      return result
    } catch (e) {
      console.error('Delegator Function was not deleted. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}
