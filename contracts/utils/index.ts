import * as fs from 'fs'
import * as path from 'path'

interface deployments {
  [network: string]: {
    [contractName: string]: string
  }
}

const deploymentsPath = path.resolve(__dirname, '..') + '/deployments/'

const isValidPath = (_path: string): boolean => {
  //checks if json file depth is 2, which stands for /{network}/{contract}.json
  const splitted = _path.replace(deploymentsPath, '').split('/')
  return splitted.length == 2
}

const readDeployments = async (folderPath: string): Promise<deployments> => {
  const result: deployments = {}

  const readFolder = async (_folderPath: string): Promise<void> => {
    const files = await fs.promises.readdir(_folderPath)

    await Promise.all(
      files.map(async (file) => {
        const filePath = path.join(_folderPath, file)
        const stats = await fs.promises.stat(filePath)

        if (stats.isDirectory()) {
          await readFolder(filePath)
        } else if (path.extname(file) === '.json' && isValidPath(filePath)) {
          const network = filePath.replace(deploymentsPath, '').split('/')[0]
          const fileContent = await fs.promises.readFile(filePath, 'utf-8')

          let contractName = path.basename(file, '.json')
          if (contractName.split('_').length > 1) {
            // remove last part which normally holds version name
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
  const contractAddressObject = await readDeployments(deploymentsPath)
  const contracts = contractAddressObject[network]
  return contracts[contractName] || ''
}

export const getAggregatorAddress = async (network: string, token_0: string, token_1: string) => {
  const contractAddressObject = await readDeployments(deploymentsPath)
  const contracts = contractAddressObject[network]

  let result = ''
  Object.keys(contracts).forEach((contractName) => {
    if (
      contractName.toLowerCase().startsWith('aggregator_') &&
      contractName.toLowerCase().includes(token_0.toLowerCase()) &&
      contractName.toLowerCase().includes(token_1.toLowerCase())
    ) {
      result = contracts[contractName]
      return
    }
  })
  return result
}

export const getAggregatorProxyAddress = async (
  network: string,
  token_0: string,
  token_1: string
) => {
  const contractAddressObject = await readDeployments(deploymentsPath)
  const contracts = contractAddressObject[network]
  let result = ''
  Object.keys(contracts).forEach((contractName) => {
    if (
      contractName.toLowerCase().startsWith('aggregatorproxy_') &&
      contractName.toLowerCase().includes(token_0.toLowerCase()) &&
      contractName.toLowerCase().includes(token_1.toLowerCase())
    ) {
      result = contracts[contractName]
      return
    }
  })
  return result
}
