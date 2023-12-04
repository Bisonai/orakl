import * as fs from 'fs'
import * as path from 'path'

const deploymentsPath = path.resolve(__dirname, '..') + '/deployments/'

const isValidPath = (_path: string): boolean => {
  const splitted = _path.replace(deploymentsPath, '').split('/')
  return splitted.length == 2
}

const readJsonFilesRecursively = async (
  folderPath: string
): Promise<{
  [network: string]: {
    [contractName: string]: string
  }
}> => {
  const result: {
    [network: string]: {
      [contractName: string]: string
    }
  } = {}

  const readFolder = async (currentFolderPath: string): Promise<void> => {
    const files = await fs.promises.readdir(currentFolderPath)

    await Promise.all(
      files.map(async (file) => {
        const filePath = path.join(currentFolderPath, file)
        // console.log(filePath)
        const stats = await fs.promises.stat(filePath)

        if (stats.isDirectory()) {
          // If it's a directory, recursively read its content
          await readFolder(filePath)
        } else if (path.extname(file) === '.json' && isValidPath(filePath)) {
          const network = filePath.replace(deploymentsPath, '').split('/')[0]

          // If it's a JSON file, read and parse it
          const fileContent = await fs.promises.readFile(filePath, 'utf-8')
          let contractName = path.basename(file, '.json')
          if (contractName.split('_').length > 1) {
            const splitted = contractName.split('_')
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
          } catch (error: any) {
            console.error(`Error parsing JSON file ${file}: ${error.message}`)
          }
        }
      })
    )
  }

  await readFolder(folderPath)
  return result
}

export const getContractAddressExact = async (network: string, contractName: string) => {
  const contractAddressObject = await readJsonFilesRecursively(deploymentsPath)
  const contracts = contractAddressObject[network]
  return contracts[contractName] || ''
}

export const getAggregatorAddress = async (network: string, pair_0: string, pair_1: string) => {
  const contractAddressObject = await readJsonFilesRecursively(deploymentsPath)
  const contracts = contractAddressObject[network]

  let result = ''
  Object.keys(contracts).forEach((contractName) => {
    if (
      contractName.toLowerCase().startsWith('aggregator_') &&
      contractName.toLowerCase().includes(pair_0.toLowerCase()) &&
      contractName.toLowerCase().includes(pair_1.toLowerCase())
    ) {
      result = contracts[contractName]
      return
    }
  })
  return result
}

export const getAggregatorProxyAddress = async (
  network: string,
  pair_0: string,
  pair_1: string
) => {
  const contractAddressObject = await readJsonFilesRecursively(deploymentsPath)
  const contracts = contractAddressObject[network]
  let result = ''
  Object.keys(contracts).forEach((contractName) => {
    if (
      contractName.toLowerCase().startsWith('aggregatorproxy_') &&
      contractName.toLowerCase().includes(pair_0.toLowerCase()) &&
      contractName.toLowerCase().includes(pair_1.toLowerCase())
    ) {
      result = contracts[contractName]
      return
    }
  })
  return result
}
