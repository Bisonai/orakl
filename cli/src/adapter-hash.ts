// import {
//   command,
//   subcommands,
//   option,
//   string as cmdstring,
//   boolean as cmdboolean,
//   flag
// } from 'cmd-ts'
// import { computeDataHash } from './utils'
// import { ReadFile, IAdapter } from './cli-types'
//
// export function adapterHashSub() {
//   // adapter
//
//   const computeAdapterHash = command({
//     name: 'compute',
//     args: {
//       verify: flag({
//         type: cmdboolean,
//         long: 'verify'
//       }),
//       adapter: option({
//         type: ReadFile,
//         long: 'file-path'
//       })
//     },
//     handler: adapterHashHandler()
//   })
//
//   return subcommands({
//     name: 'adapterHash',
//     cmds: { computeAdapterHash }
//   })
// }
//
//
