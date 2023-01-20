import { command, subcommands, option, string as cmdstring } from 'cmd-ts'
import { chainOptionalOption, dryrunOption, idOption } from './utils'

export function kvCmd(db) {
  // yarn cli kv list  --chain [chain]
  // yarn cli kv insert --key PUBLIC_KEY --value HELLO --chain localhost
  // yarn cli kv remove --key PUBLIC_KEY --value HELLO --chain localhost
  // yarn cli kv update --key PUBLIC_KEY --value HELLO --chain localhost

  const list = command({
    name: 'list',
    args: {
      chain: chainOptionalOption
    },
    handler: listHandler(db)
  })

  return subcommands({
    name: 'kv',
    cmds: { list }
  })
}

export function listHandler(db) {
  async function wrapper() {
    const query = 'SELECT * FROM Kv'
    const result = await db.all(query)
    console.log(result)
    return result
  }
  return wrapper
}
