const fs = require('fs')

// Get the key from the terminal argument
const hash = process.argv[2]
const MAPPING_PATH = 'scripts/sol-error-mappings/errorMappings.json'

function main() {
  if (!hash) {
    console.error('hash not provided')
    process.exit(1)
  }

  try {
    const data = fs.readFileSync(MAPPING_PATH, 'utf8')
    const obj = JSON.parse(data)
    const value = obj[hash]
    if (!value) {
      console.error('----hash not found----')
    } else {
      console.log(value)
    }
  } catch (error) {
    console.error(error)
    process.exit(1)
  }
}

main()
