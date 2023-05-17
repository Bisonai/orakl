import { createCipheriv, createDecipheriv, randomBytes, scrypt } from 'crypto'
import { promisify } from 'util'

export async function flattenReporter(L) {
  return {
    id: L?.id,
    address: L?.address,
    privateKey: await decryptText(L?.privateKey),
    oracleAddress: L?.oracleAddress,
    service: L?.service.name,
    chain: L?.chain?.name
  }
}
export async function encryptText(textToEncrypt: string) {
  const password = process.env.ENCRYPT_PASSWORD || 'bisonai@123'
  const iv = randomBytes(16).toString('hex')
  // The key length is dependent on the algorithm.
  // In this case for aes256, it is 32 bytes.
  const key = (await promisify(scrypt)(password, 'salt', 32)) as Buffer
  const cipher = createCipheriv('aes-256-ctr', key, Buffer.from(iv, 'hex'))
  const encryptedText = Buffer.concat([cipher.update(textToEncrypt), cipher.final()])
  return `${iv}${encryptedText.toString('hex')}`
}

export async function decryptText(encryptedText: string) {
  const password = process.env.ENCRYPT_PASSWORD
  const iv = encryptedText.substring(0, 32)
  const textToDecrypt = encryptedText.substring(32, encryptedText.length)
  const key = (await promisify(scrypt)(password, 'salt', 32)) as Buffer
  const decipher = createDecipheriv('aes-256-ctr', key, Buffer.from(iv, 'hex'))
  const decryptedText = Buffer.concat([decipher.update(textToDecrypt, 'hex'), decipher.final()])
  return decryptedText.toString()
}
