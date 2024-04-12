const path = require('path')
const fs = require('fs')
const { loadJson, storeJson } = require('./utils.cjs')

const ValidChains = ['baobab', 'cypress']
const deploymentsPath = path.join(__dirname, '../deployments/')

const isValidPath = (_path) => {
  const fileName = path.basename(_path)
  return path.extname(fileName) === '.json'
}

const readDeployments = async (folderPath) => {
  const dataFeeds = {}
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
          const network = filePath.replace(deploymentsPath, '').split('/')[0]
          if (!ValidChains.includes(network)) {
            return
          }

          const address = await readAddressFromFile(filePath)
          if (!address) {
            return
          }

          let contractName = path.basename(file, '.json')

          if (contractName.startsWith('Aggregator') && contractName.includes('_')) {
            const splitted = contractName.replace(' ', '').split('_')
            const contractType = splitted[0]
            const pairName = splitted[1]

            if (!dataFeeds[pairName]) {
              dataFeeds[pairName] = {}
            }
            if (!dataFeeds[pairName][network]) {
              dataFeeds[pairName][network] = {}
            }
            dataFeeds[pairName][network][convertContractType(contractType)] = address
          } else {
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
        }
      })
    )
  }

  try {
    await readFolder(folderPath)
  } catch (error) {
    console.error(`Failed to read directory: ${error}`)
  }

  return { dataFeeds, others }
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

const convertContractType = (contractType) => {
  if (contractType == 'Aggregator') {
    return 'aggregator'
  } else if (contractType == 'AggregatorProxy') {
    return 'proxy'
  } else {
    return 'unknown'
  }
}

async function main() {
  const { dataFeeds, others } = await readDeployments(deploymentsPath)
  await storeJson(
    path.join(deploymentsPath, 'datafeeds-addresses.json'),
    JSON.stringify(dataFeeds, null, 2)
  )
  await storeJson(
    path.join(deploymentsPath, 'other-addresses.json'),
    JSON.stringify(others, null, 2)
  )
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
