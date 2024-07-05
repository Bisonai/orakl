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
  return splitted.length == 2 && path.extname(_path) === '.json'
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
        } else if (isValidPath(filePath)) {
          const network = filePath.replace(deploymentsPath, '').split('/')[0]
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
          } catch (error: any) {
            console.error(`Error parsing JSON file ${file}: ${error.message}`)
          }
        }
      }),
    )
  }

  await readFolder(folderPath)
  return result
}

export const getContractAddress = async (network: string, contractName: string) => {
  const contractAddressObject = await readDeployments(deploymentsPath)
  const contracts = contractAddressObject[network]
  return contracts[contractName] || ''
}

const _getContractAddressWithTokenPairs = async (
  network: string,
  contractName: string,
  token_0: string,
  token_1: string,
) => {
  const name = `${contractName}_${token_0.toUpperCase()}-${token_1.toUpperCase()}`
  return await getContractAddress(network, name)
}

export const getAggregatorAddress = async (network: string, token_0: string, token_1: string) => {
  return await _getContractAddressWithTokenPairs(network, 'Aggregator', token_0, token_1)
}

export const getAggregatorProxyAddress = async (
  network: string,
  token_0: string,
  token_1: string,
) => {
  return await _getContractAddressWithTokenPairs(network, 'AggregatorProxy', token_0, token_1)
}

export const getPorAddress = async (network: string, token_0: string, token_1: string) => {
  return _getContractAddressWithTokenPairs(network, 'POR', token_0, token_1)
}

export const getPorProxyAddress = async (network: string, token_0: string, token_1: string) => {
  return _getContractAddressWithTokenPairs(network, 'PORProxy', token_0, token_1)
}
