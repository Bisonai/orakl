import axios from 'axios'
import { command, option, subcommands, string as cmdstring, number as cmdnumber } from 'cmd-ts'
import { buildUrl, idOption, isOraklDelegatorHealthy } from './utils'
import { ORAKL_NETWORK_DELEGATOR_URL } from './settings'

export function delegatorSub() {
  // delegator organizationList
  // delegator organizationInsert --name {organizationName}
  // delegator organizationRemove --id {organizationId}

  // delegator reporterList
  // delegator reporterInsert --address {reporterAddress} --organizationId {organizationId}
  // delegator reporterRemove --id {reporterId}

  // delegator contractList
  // delegator contractInsert --address {contractAddress}
  // delegator contractRemove --id {reporterId}

  // delegator functionList
  // delegator functionInsert --address {contractAddress} --organizationId {organizationId}
  // delegator functionRemove --id {reporterId}

  const organizationList = command({
    name: 'organizationList',
    args: {},
    handler: organizationListHandler()
  })

  const organizationInsert = command({
    name: 'organizationInsert',
    args: {
      name: option({
        type: cmdstring,
        long: 'name'
      })
    },
    handler: organizationInsertHandler()
  })

  const organizationRemove = command({
    name: 'organizationRemove',
    args: {
      id: idOption
    },
    handler: organizationRemoveHandler()
  })

  const reporterList = command({
    name: 'reporterList',
    args: {},
    handler: reporterListHandler()
  })

  const reporterInsert = command({
    name: 'reporterInsert',
    args: {
      address: option({
        type: cmdstring,
        long: 'address'
      }),
      organizationId: option({
        type: cmdnumber,
        long: 'organizationId'
      })
    },
    handler: reporterInsertHandler()
  })

  const reporterRemove = command({
    name: 'reporterRemove',
    args: {
      id: idOption
    },
    handler: reporterRemoveHandler()
  })

  const contractList = command({
    name: 'contractList',
    args: {},
    handler: contractListHandler()
  })

  const contractInsert = command({
    name: 'contractInsert',
    args: {
      address: option({
        type: cmdstring,
        long: 'address'
      })
    },
    handler: contractInsertHandler()
  })

  const contractRemove = command({
    name: 'contractRemove',
    args: {
      id: idOption
    },
    handler: contractRemoveHandler()
  })

  const functionList = command({
    name: 'functionList',
    args: {},
    handler: functionListHandler()
  })

  const functionInsert = command({
    name: 'functionInsert',
    args: {
      name: option({
        type: cmdstring,
        long: 'name'
      }),
      contractId: option({
        type: cmdnumber,
        long: 'contractId'
      }),
      encodedName: option({
        type: cmdstring,
        long: 'encodedName'
      })
    },
    handler: functionInsertHandler()
  })

  const functionRemove = command({
    name: 'functionRemove',
    args: {
      id: idOption
    },
    handler: functionRemoveHandler()
  })

  return subcommands({
    name: 'delegator',
    cmds: {
      organizationList,
      organizationInsert,
      organizationRemove,
      reporterList,
      reporterInsert,
      reporterRemove,
      contractList,
      contractInsert,
      contractRemove,
      functionList,
      functionInsert,
      functionRemove
    }
  })
}

export function organizationListHandler() {
  async function wrapper() {
    if (!(await isOraklDelegatorHealthy())) return

    try {
      const endpoint = buildUrl(ORAKL_NETWORK_DELEGATOR_URL, `organization`)
      const result = await axios.get(endpoint)
      console.log(result?.data)
    } catch (e) {
      console.error('Delegator Orgzanization was not listed. Reason:')
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
      const result = await axios.post(endpoint, { name })
      console.log(result?.data)
    } catch (e) {
      console.error('Delegator Orgzanization was not inserted. Reason:')
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
        id.toString()
      )
      const result = await axios.delete(endpoint)
      console.log(result?.data)
    } catch (e) {
      console.error('Delegator Orgzanization was not deleted. Reason:')
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
      const result = await axios.get(endpoint)
      console.log(result?.data)
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
      const result = await axios.post(endpoint, { address, organizationId })
      console.log(result?.data)
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
      const result = await axios.delete(endpoint)
      console.log(result?.data)
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
      const result = await axios.get(endpoint)
      console.log(result?.data)
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
      const result = await axios.post(endpoint, { address })
      console.log(result?.data)
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
      const result = await axios.delete(endpoint)
      console.log(result?.data)
    } catch (e) {
      console.error('Delegator Contract was not deleted. Reason:')
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
      const result = await axios.get(endpoint)
      console.log(result?.data)
    } catch (e) {
      console.error('Delegator Function was not listed. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function functionInsertHandler() {
  async function wrapper({
    name,
    contractId,
    encodedName
  }: {
    name: string
    contractId: number
    encodedName: string
  }) {
    if (!(await isOraklDelegatorHealthy())) return

    try {
      const endpoint = buildUrl(ORAKL_NETWORK_DELEGATOR_URL, `function`)
      const result = await axios.post(endpoint, { name, contractId, encodedName })
      console.log(result?.data)
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
      const result = await axios.delete(endpoint)
      console.log(result?.data)
    } catch (e) {
      console.error('Delegator Function was not deleted. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}
