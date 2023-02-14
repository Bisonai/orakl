import { flag, option, optional, command, string as cmdstring, boolean as cmdboolean } from 'cmd-ts'

export function migrateCmd(db) {
  // migrate [--force] [--migrationspath [path]]

  return command({
    name: 'migrate',
    args: {
      force: flag({
        type: cmdboolean,
        long: 'force'
      }),
      migrationsPath: option({
        type: optional(cmdstring),
        long: 'migrationsPath'
      })
    },
    handler: async ({ force, migrationsPath }) => {
      await db.migrate({ force, migrationsPath })
    }
  })
}
