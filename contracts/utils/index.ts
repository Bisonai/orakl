import * as fs from 'fs'
import * as path from 'path'

const projectRootPath = path.resolve(__dirname, '..')

const readJsonFilesRecursively = async (folderPath: string): Promise<{ [key: string]: any }> => {
  const result: { [key: string]: any } = {}

  const readFolder = async (currentFolderPath: string): Promise<void> => {
    const files = await fs.promises.readdir(currentFolderPath)

    await Promise.all(
      files.map(async (file) => {
        const filePath = path.join(currentFolderPath, file)
        const stats = await fs.promises.stat(filePath)

        if (stats.isDirectory()) {
          // If it's a directory, recursively read its content
          await readFolder(filePath)
        } else if (path.extname(file) === '.json') {
          console.log(filePath)
          // If it's a JSON file, read and parse it
          const fileContent = await fs.promises.readFile(filePath, 'utf-8')
          const fileNameWithoutExtension = path.basename(file, '.json')

          try {
            result[fileNameWithoutExtension] = JSON.parse(fileContent)
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

// const getAddress = (network: string, contract: string): string => {

// }

const main = async () => {
  const result = await readJsonFilesRecursively(projectRootPath + '/deployments/')
  // console.log(result)
}

main()
