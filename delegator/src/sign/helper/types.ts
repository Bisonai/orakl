import { SignatureData } from 'caver-js'

export interface SignTxData {
  from: string
  to: string
  input: string
  gas: string
  signatures: SignatureData[] | SignatureData
  value: string
  chainId: string
  gasPrice: string
  nonce: string
}
