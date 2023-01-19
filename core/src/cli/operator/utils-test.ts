import { open } from 'sqlite'
import { Database } from 'sqlite3'
import { SETTINGS_DB_FILE } from '../../settings'

export async function openDb({ migrate }: { migrate?: boolean }) {
  const db = await open({
    filename: SETTINGS_DB_FILE,
    driver: Database
  })

  if (migrate) {
    await db.migrate({ force: true })
  }

  return db
}
