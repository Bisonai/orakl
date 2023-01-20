import fs from 'fs'
import { SETTINGS_DB_FILE } from './settings'
import sqlite from 'sqlite3'
import { open } from 'sqlite'

export async function dbChecker() {
  if (!fs.existsSync(SETTINGS_DB_FILE)) {
    await open({
      filename: SETTINGS_DB_FILE,
      driver: sqlite.Database
    }).then(async (db) => {
      const { count } = await db.get(
        `SELECT count(*) AS count FROM sqlite_master WHERE type='table'`
      )
      if (count < 1) {
        await db.migrate()
      }
    })
  }
}
