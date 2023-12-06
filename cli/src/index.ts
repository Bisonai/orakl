#!/usr/bin/env node --no-warnings --experimental-specifier-resolution=node

import { binary, command, run, subcommands } from 'cmd-ts'
import { adapterSub } from './adapter'
import { aggregatorSub } from './aggregator'
import { chainSub } from './chain'
import { datafeedSub } from './datafeed'
import { delegatorSub } from './delegator'
import { fetcherSub } from './fetcher'
import { listenerSub } from './listener'
import { proxySub } from './proxy'
import { reporterSub } from './reporter'
import { serviceSub } from './service'
import { vrfSub } from './vrf'

async function main() {
  const chain = chainSub()
  const service = serviceSub()
  const listener = listenerSub()
  const vrf = vrfSub()
  const adapter = adapterSub()
  const aggregator = aggregatorSub()
  const fetcher = fetcherSub()
  const reporter = reporterSub()
  const delegator = delegatorSub()
  const proxy = proxySub()
  const datafeed = datafeedSub()

  const version = command({
    name: 'version',
    args: {},
    handler: () => {
      console.log(`Orakl Network CLI v${process.env.npm_package_version}`)
    }
  })

  const cli = subcommands({
    name: 'operator',
    cmds: {
      chain,
      service,
      listener,
      vrf,
      adapter,
      aggregator,
      fetcher,
      reporter,
      version,
      delegator,
      proxy,
      datafeed
    }
  })

  run(binary(cli), process.argv)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
