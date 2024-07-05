const fs = require('node:fs')
const path = require('node:path')
const sha3 = require('js-sha3')

const PROCESS_PATH = 'src'
const RESULT_PATH = 'scripts/sol-error-mappings/errorMappings.json'

function processFile(filePath) {
  const content = fs.readFileSync(filePath, 'utf8')
  const lines = content.split('\n')
  const mapping = {}

  for (const line of lines) {
    const trimmedLine = line.trim()
    if (trimmedLine.startsWith('error ')) {
      const processedLine = trimmedLine
        .replace('error ', '')
        .replace(
          /\(([^)]+)\)/,
          (match) =>
            `${match
              .split(' ')
              .filter((_, idx) => idx % 2 === 0)
              .join(',')})`,
        )
        .replace(';', '')
      const hash = `0x${sha3.keccak256(processedLine)}`
      mapping[hash] = {
        file: filePath,
        error: trimmedLine,
      }
    }
  }

  return mapping
}

function processDirectory(dirPath) {
  const entries = fs.readdirSync(dirPath, { withFileTypes: true })
  let mapping = {}

  for (const entry of entries) {
    const fullPath = path.join(dirPath, entry.name)
    if (entry.isDirectory()) {
      mapping = { ...mapping, ...processDirectory(fullPath) }
    } else if (entry.isFile() && path.extname(entry.name) === '.sol') {
      mapping = { ...mapping, ...processFile(fullPath) }
    }
  }

  return mapping
}

const mapping = processDirectory(PROCESS_PATH)
fs.writeFileSync(RESULT_PATH, JSON.stringify(mapping, null, 2))
