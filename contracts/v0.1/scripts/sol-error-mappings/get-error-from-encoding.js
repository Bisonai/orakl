const fs = require('node:fs')

// Get the key from the terminal argument
const hash = process.argv[2]
const MAPPING_PATH = 'scripts/sol-error-mappings/errorMappings.json'

function handleLogging({ exitCode, logMessage, errorMessage }) {
  if (logMessage) {
    console.log(logMessage)
  }
  if (errorMessage) {
    console.error(errorMessage)
  }
  if (exitCode) {
    process.exit(exitCode)
  }
}

function main() {
  if (!hash) {
    handleLogging({ exitCode: 1, errorMessage: 'hash not provided' })
  }

  try {
    const data = fs.readFileSync(MAPPING_PATH, 'utf8')
    const obj = JSON.parse(data)
    const value = obj[hash]
    if (!value) {
      handleLogging({ exitCode: 1, errorMessage: 'hash not found' })
    } else {
      handleLogging({ logMessage: value })
    }
  } catch (error) {
    handleLogging({ exitCode: 1, errorMessage: error.message })
  }
}

main()
