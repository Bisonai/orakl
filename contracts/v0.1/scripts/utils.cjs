const { readdir, readFile, appendFile, writeFile } = require('node:fs/promises')
const path = require('node:path')
const moment = require('moment')
const { writeFileSync } = require('node:fs')
const MIGRATION_LOCK_FILE_NAME = 'migration.lock'

async function loadJson(filepath) {
  try {
    const json = await readFile(filepath, 'utf8')
    return JSON.parse(json)
  } catch (e) {
    console.error(e)
    throw e
  }
}

async function storeJson(filepath, data) {
  try {
    await writeFileSync(filepath, data)
  } catch (e) {
    console.error(e)
    throw e
  }
}

async function readMigrationLockFile(filePath) {
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
async function loadMigration(dirPath) {
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
 * @params {string} migration directory
 * @params {string} name of executed migration file that should be included to migration lock file
 * @return {Promise<void>}
 */
async function updateMigration(dirPath, migrationFileName) {
  const migrationLockFilePath = path.join(dirPath, MIGRATION_LOCK_FILE_NAME)
  await appendFile(migrationLockFilePath, `${migrationFileName}\n`)
}

function validateProperties(config, requiredProperties) {
  for (const rp of requiredProperties) {
    if (config[rp] === undefined) return false
  }

  return true
}

/**
 * @params {IAggregatorDeployConfig}
 * @return {boolean}
 */
function validateAggregatorDeployConfig(config) {
  const requiredProperties = ['name', 'timeout', 'validator', 'decimals', 'description']

  if (!validateProperties(config, requiredProperties)) return false

  if (config.paymentAmount > 0 && config.depositAmount && config.depositAmount > 0) {
    return false
  }

  return true
}

/**
 * @params {IAggregatorChangeOraclesConfig}
 * @return {boolean}
 */
function validateAggregatorChangeOraclesConfig(config) {
  const requiredProperties = [
    'removed',
    'added',
    'minSubmissionCount',
    'maxSubmissionCount',
    'restartDelay',
  ]

  if (!validateProperties(config, requiredProperties)) {
    return false
  } else {
    return true
  }
}

/**
 * @params {ICoordinatorDeploy}
 * @return {boolean}
 */
function validateCoordinatorDeployConfig(config) {
  const requiredProperties = ['version']

  if (!validateProperties(config, requiredProperties)) {
    return false
  } else {
    return true
  }
}

/**
 * @params {ICoordinatorConfig}
 * @return {boolean}
 */
function validateSetConfig(config) {
  const requiredProperties = ['maxGasLimit', 'gasAfterPaymentCalculation', 'feeConfig']

  if (!validateProperties(config, requiredProperties)) {
    return false
  } else {
    return true
  }
}

/**
 * @params {IRegisterOracle[]}
 * @return {boolean}
 */
function validateVrfRegisterOracle(config) {
  const requiredProperties = ['address', 'publicProvingKey']

  for (const c of config) {
    if (!validateProperties(c, requiredProperties)) {
      return false
    }
  }

  return true
}

/**
 * @params {IDeregisterOracle[]}
 * @return {boolean}
 */
function validateVrfDeregisterOracle(config) {
  const requiredProperties = ['address']

  for (const c of config) {
    if (!validateProperties(c, requiredProperties)) {
      return false
    }
  }

  return true
}

/**
 * @params {IPrepaymentDeploy}
 * @return {boolean}
 */
function validatePrepaymentDeployConfig(config) {
  const requiredProperties = ['protocolFeeRecipient']

  if (!validateProperties(config, requiredProperties)) {
    return false
  } else {
    return true
  }
}

function validateAggregatorRedirectProxyConfig(config) {
  const requiredProperties = ['status', 'proxyAddress', 'aggregator']

  if (!validateProperties(config, requiredProperties)) {
    return false
  } else {
    return true
  }
}

function getFormattedDate() {
  return moment().format('YYYYMMDDHHMMSS')
}

// returns object with {key(contractName):value(address)} inside dirPath
async function loadDeployments(dirPath) {
  const jsonFileRegex = /\.json$/
  const result = {}
  try {
    const files = await readdir(dirPath)

    for (const file of files) {
      if (jsonFileRegex.test(file.toLowerCase())) {
        let contractName = path.basename(file, '.json')
        if (contractName.split('_').length > 1) {
          // remove last part which normally holds version name
          const splitted = contractName.replace(' ', '').split('_')
          splitted.pop()
          contractName = splitted.join('_')
        }
        const filePath = path.join(dirPath, file)
        const deploymentDetail = await loadJson(filePath)
        const address = deploymentDetail.address
        if (!address) {
          continue
        }
        result[contractName] = address
      }
    }
  } catch (err) {
    console.error(err)
  }

  return result
}

module.exports = {
  loadJson,
  storeJson,
  loadMigration,
  updateMigration,
  validateAggregatorDeployConfig,
  validateAggregatorChangeOraclesConfig,
  validateCoordinatorDeployConfig,
  validateSetConfig,
  validateVrfRegisterOracle,
  validateVrfDeregisterOracle,
  validatePrepaymentDeployConfig,
  validateAggregatorRedirectProxyConfig,
  getFormattedDate,
  loadDeployments,
}
