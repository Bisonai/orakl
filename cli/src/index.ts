#!/usr/bin/env node --no-warnings

import { binary, command, run, subcommands } from 'cmd-ts'
import { chainSub } from './chain.js'
import { delegatorSub } from './delegator.js'
import { listenerSub } from './listener.js'
import { proxySub } from './proxy.js'
import { reporterSub } from './reporter.js'
import { serviceSub } from './service.js'
import { vrfSub } from './vrf.js'

async function main() {
  const chain = chainSub()
  const service = serviceSub()
  const listener = listenerSub()
  const vrf = vrfSub()
  const reporter = reporterSub()
  const delegator = delegatorSub()
  const proxy = proxySub()

  const version = command({
    name: 'version',
    args: {},
    handler: () => {
      console.log(`Orakl Network CLI v${process.env.npm_package_version}`)
    },
  })

  const cli = subcommands({
    name: 'operator',
    cmds: {
      chain,
      service,
      listener,
      vrf,
      reporter,
      version,
      delegator,
      proxy,
    },
  })

  run(binary(cli), process.argv)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
