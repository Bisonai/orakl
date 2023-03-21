import { readdir, readFile, appendFile } from 'node:fs/promises'
import * as path from 'node:path'
import {
  IRRCSetMinBalance,
  IRRCDeploy,
  IAggregatorDeployConfig,
  IAggregatorChangeOraclesConfig,
  IVrfDeploy,
  IRegisterProvingKey,
  IDeregisterProvingKey,
  ICoordinatorConfig,
  ICoordinatorDirectPaymentConfig
} from './types'

const MIGRATION_LOCK_FILE_NAME = 'migration.lock'

export async function loadJson(filepath: string) {
  try {
    const json = await readFile(filepath, 'utf8')
    return JSON.parse(json)
  } catch (e) {
    console.error(e)
    raise(e)
  }
}

async function readMigrationLockFile(filePath: string) {
  return (await readFile(filePath, 'utf8')).toString().trim().split('\n')
}

/**
 * Migrations directory includes migration JSON files and migration lock file.
 * Migration JSON files should have a names that explains the
 * migration purpose and which can be chronological order. If there is
 * no migration lock file, we assume that no migration has been run
 * yet. `loadMigration` function examines migration JSON files,
 * migration lock file and determines which migration JSON files
 * should be used for next migration.
 *
 * @param {string} migrations directory
 * @return {Promise<string[]>} list of migration files names that has
 * not been applied yet
 */
export async function loadMigration(dirPath: string): Promise<string[]> {
  const jsonFileRegex = /\.json$/

  let migrationLockFileExist = false
  const allMigrations = []

  try {
    const files = await readdir(dirPath)

    for (const file of files) {
      if (file === MIGRATION_LOCK_FILE_NAME) {
        migrationLockFileExist = true
      } else if (jsonFileRegex.test(file.toLowerCase())) {
        allMigrations.push(file)
      }
    }
  } catch (err) {
    console.error(err)
  }

  let doneMigrations = []
  if (migrationLockFileExist) {
    const migrationLockFilePath = path.join(dirPath, MIGRATION_LOCK_FILE_NAME)
    doneMigrations = await readMigrationLockFile(migrationLockFilePath)
  }

  // Keep only those migrations that have not been applied yet
  const todoMigrations = allMigrations.filter((x) => !doneMigrations.includes(x))
  todoMigrations.sort()
  return todoMigrations
}

/**
 * Update migration lock file located in `dirPath` with the `migrationFileName` migration.
 *
 * @params {dirPath} migration directory
 * @params {migrationFileName} name of executed migration file that should be included to migration lock file
 * @return {void}
 */
export async function updateMigration(dirPath: string, migrationFileName: string) {
  const migrationLockFilePath = path.join(dirPath, MIGRATION_LOCK_FILE_NAME)
  await appendFile(migrationLockFilePath, `${migrationFileName}\n`)
}

function validateProperties(config, requiredProperties: string[]) {
  for (const rp of requiredProperties) {
    if (config[rp] === undefined) return false
  }

  return true
}

export function validateAggregatorDeployConfig(config: IAggregatorDeployConfig): boolean {
  const requiredProperties = [
    'name',
    'paymentAmount',
    'timeout',
    'validator',
    'decimals',
    'description'
  ]

  if (!validateProperties(config, requiredProperties)) return false

  if (config.paymentAmount > 0 && config.depositAmount > 0) {
    return false
  }

  return true
}

export function validateAggregatorChangeOraclesConfig(
  config: IAggregatorChangeOraclesConfig
): boolean {
  const requiredProperties = [
    'removed',
    'added',
    'addedAdmins',
    'minSubmissionCount',
    'maxSubmissionCount',
    'restartDelay'
  ]

  if (!validateProperties(config, requiredProperties)) {
    return false
  } else {
    return true
  }
}

export function validateRRCDeployConfig(config: IRRCDeploy): boolean {
  const requiredProperties = ['version']

  if (!validateProperties(config, requiredProperties)) {
    return false
  } else {
    return true
  }
}

export function validateMinBalanceConfig(config: IRRCSetMinBalance): boolean {
  const requiredProperties = ['minBalance']

  if (!validateProperties(config, requiredProperties)) {
    return false
  } else {
    return true
  }
}

export function validateSetConfig(config: ICoordinatorConfig): boolean {
  const requiredProperties = ['maxGasLimit', 'gasAfterPaymentCalculation', 'feeConfig']

  if (!validateProperties(config, requiredProperties)) {
    return false
  } else {
    return true
  }
}

export function validateDirectPaymentConfig(config: ICoordinatorDirectPaymentConfig): boolean {
  if (!validateProperties(config, ['directPaymentConfig'])) {
    return false
  }

  if (!validateProperties(config.directPaymentConfig, ['fulfillmentFee', 'baseFee'])) {
    return false
  }

  return true
}

export function validateVrfDeployConfig(config: IVrfDeploy): boolean {
  const requiredProperties = ['version']

  if (!validateProperties(config, requiredProperties)) {
    return false
  } else {
    return true
  }
}

export function validateVrfRegisterProvingKey(config: IRegisterProvingKey[]): boolean {
  const requiredProperties = ['address', 'publicProvingKey']

  for (const c of config) {
    if (!validateProperties(c, requiredProperties)) {
      return false
    }
  }

  return true
}

export function validateVrfDeregisterProvingKey(config: IDeregisterProvingKey[]): boolean {
  const requiredProperties = ['publicProvingKey']

  for (const c of config) {
    if (!validateProperties(c, requiredProperties)) {
      return false
    }
  }

  return true
}
