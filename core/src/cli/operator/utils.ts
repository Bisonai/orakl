import {
  optional,
  boolean as cmdboolean,
  number as cmdnumber,
  string as cmdstring,
  flag,
  option
} from 'cmd-ts'
import sqlite from 'sqlite3'
import { open } from 'sqlite'
import { SETTINGS_DB_FILE } from '../../settings'

export async function openDb() {
  return await open({
    filename: SETTINGS_DB_FILE,
    driver: sqlite.Database
  })
}

export const dryrunOption = flag({
  type: cmdboolean,
  long: 'dry-run'
})

export const idOption = option({
  type: cmdnumber,
  long: 'id'
})

export function buildStringOption({ name, isOptional }: { name: string; isOptional?: boolean }) {
  return option({
    type: isOptional ? optional(cmdstring) : cmdstring,
    long: name
  })
}
