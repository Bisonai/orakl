import { flag, command, boolean as cmdboolean } from 'cmd-ts'

export function migrateCmd(db) {
  return command({
    name: 'migrate',
    args: {
      force: flag({
        type: cmdboolean,
        long: 'force'
      })
    },
    handler: async ({ force }) => {
      await db.migrate({ force })
    }
  })
}
