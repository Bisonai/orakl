const path = require('path')
const fs = require('fs')
const axios = require('axios')
const { loadJson, storeJson } = require('./utils.cjs')

const ValidChains = ['baobab', 'cypress']
const deploymentsPath = path.join(__dirname, '../deployments/')

const isValidPath = (_path) => {
  const fileName = path.basename(_path)
  return path.extname(fileName) === '.json'
}

const readDeployments = async (folderPath) => {
  const others = {}

  const readFolder = async (_folderPath) => {
    const files = await fs.promises.readdir(_folderPath)

    await Promise.all(
      files.map(async (file) => {
        const filePath = path.join(_folderPath, file)
        const stats = await fs.promises.stat(filePath)

        if (stats.isDirectory()) {
          await readFolder(filePath)
        } else if (isValidPath(filePath)) {
          if (file.startsWith('Aggregator_') || file.startsWith('AggregatorProxy_')) {
            return
          }
          const network = filePath.replace(deploymentsPath, '').split('/')[0]
          if (!ValidChains.includes(network)) {
            return
          }

          const address = await readAddressFromFile(filePath)
          if (!address) {
            return
          }

          let contractName = path.basename(file, '.json')

          if (!others[network]) {
            others[network] = {}
          }
          const splitted = contractName.replace(' ', '').split('_')
          if (splitted.length > 1) {
            splitted.pop()
            contractName = splitted.join('_')
          }
          others[network][contractName] = address
        }
      }),
    )
  }

  try {
    await readFolder(folderPath)
  } catch (error) {
    console.error(`Failed to read directory: ${error}`)
  }

  return others
}

async function readAddressFromFile(filePath) {
  try {
    const fileContent = await loadJson(filePath)
    return fileContent.address
  } catch (error) {
    console.error(`Error reading file ${filePath}: ${error.message}`)
    return null
  }
}

async function main() {
  const others = await readDeployments(deploymentsPath)

  await storeJson(
    path.join(deploymentsPath, 'other-addresses.json'),
    JSON.stringify(others, null, 2),
  )
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
