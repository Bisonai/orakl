const path = require('path')
const fs = require('fs')
const { storeJson } = require('./utils.cjs')

const ValidChains = ['baobab', 'cypress']
const deploymentsPath = path.join(__dirname, '../deployments/')

const isValidPath = (_path) => {
  const splitted = _path.replace(deploymentsPath, '').split('/')
  return splitted.length == 2 && path.extname(_path) === '.json'
}

const readDeployments = async (folderPath) => {
  const result = {}

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
          const fileContent = await fs.promises.readFile(filePath, 'utf-8')

          let contractName = path.basename(file, '.json')
          if (contractName.split('_').length > 1) {
            // remove last part which normally holds version name
            const splitted = contractName.replace(' ', '').split('_')
            splitted.pop()
            contractName = splitted.join('_')
          }

          try {
            const jsonObject = JSON.parse(fileContent)
            const contractAddress = jsonObject.address
            if (!contractAddress) {
              return
            }

            result[network] = { ...result[network], [contractName]: contractAddress }
          } catch (error) {
            console.error(`Error parsing JSON file ${file}: ${error.message}`)
          }
        }
      })
    )
  }

  await readFolder(folderPath)
  return result
}

async function main() {
  const storeFilePath = path.join(deploymentsPath, 'collected-addresses.json')
  const contractAddresses = await readDeployments(deploymentsPath)

  await storeJson(storeFilePath, JSON.stringify(contractAddresses, null, 2))
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
