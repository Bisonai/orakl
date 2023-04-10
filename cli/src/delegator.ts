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

  const organizationList = command({
    name: 'organization-list',
    args: {},
    handler: organizationListHandler()
  })

  const organizationInsert = command({
    name: 'organization-insert',
    args: {
      name: option({
        type: cmdstring,
        long: 'name'
      })
    },
    handler: organizationInsertHandler()
  })

  const organizationRemove = command({
    name: 'organization-insert',
    args: {
      id: idOption
    },
    handler: organizationRemoveHandler()
  })

  const reporterList = command({
    name: 'reporter-list',
    args: {},
    handler: reporterListHandler()
  })

  const reporterInsert = command({
    name: 'reporter-insert',
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
    name: 'reporter-insert',
    args: {
      id: idOption
    },
    handler: reporterRemoveHandler()
  })
  return subcommands({
    name: 'delegator',
    cmds: {
      organizationList,
      organizationInsert,
      organizationRemove,
      reporterList,
      reporterInsert,
      reporterRemove
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
