import axios from 'axios'
import { command, subcommands } from 'cmd-ts'
import { buildUrl, isOraklDelegatorHealthy } from './utils'
import { ORAKL_NETWORK_DELEGATOR_URL } from './settings'

export function delegatorSub() {
  // delegator organization

  const organization = command({
    name: 'organization',
    handler: organizationHandler(),
    args: {}
  })

  return subcommands({
    name: 'delegator',
    cmds: { organization }
  })
}

export function organizationHandler() {
  async function wrapper() {
    if (!(await isOraklDelegatorHealthy())) return

    try {
      const endpoint = buildUrl(ORAKL_NETWORK_DELEGATOR_URL, `organization`)
      const response = await axios.get(endpoint)
      console.log(response?.data)
    } catch (e) {
      console.dir(e?.response?.data, { depth: null })
    }
  }
  return wrapper
}
