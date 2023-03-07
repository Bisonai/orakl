import { Transaction } from '@prisma/client'
import Caver, { AbiItem, SingleKeyring } from 'caver-js'
import { SignDto } from '../dto/sign.dto'
import { dummyFactory } from './contract'

const PROVIDER_URL = 'https://api.baobab.klaytn.net:8651'
const caver = new Caver(PROVIDER_URL)
const feePayerKeyring = caver.wallet.keyring.createFromPrivateKey(process.env.SIGNER_PRIVATE_KEY)
caver.wallet.add(feePayerKeyring)

async function getSignedTx(txId) {
  // TODO: remove fake creation Tx, read it from DB
  const contract = new caver.contract(dummyFactory.abi as AbiItem[], dummyFactory.address)
  const keyring = caver.wallet.keyring.createFromPrivateKey(process.env.CAVER_PRIVATE_KEY)
  caver.wallet.add(keyring)

  const input = contract.methods.increament().encodeABI()
  const rawTx = caver.transaction.feeDelegatedSmartContractExecution.create({
    from: keyring.address,
    to: contract._address,
    input: input,
    gas: 90000
  })
  const newRawTx = caver.transaction.feeDelegatedSmartContractExecution.create({
    from: rawTx.from,
    to: rawTx.to,
    input: rawTx.input,
    gas: rawTx.gas,
    signatures: rawTx.signatures,
    value: rawTx.value,
    chainId: rawTx.chainId,
    gasPrice: rawTx.gasPrice,
    nonce: rawTx.nonce
  })
  await caver.wallet.sign(keyring.address, newRawTx)
  return newRawTx
}

async function signByFeePayerAndExecuteTransaction(rawTx) {
  await caver.wallet.signAsFeePayer(feePayerKeyring.address, rawTx)
  return await caver.rpc.klay.sendRawTransaction(rawTx.getRawTransaction())
}

export async function approveAndSign(input: Transaction) {
  console.log('Input', input)

  // input.signed = 'true'
  // let signDto = new SignDto()
  // signDto.tx = input.tx
  // signDto.signed = input.signed

  // console.log('input:', input)
  // console.log('signDto:', signDto)

  // readTxFromDb
  const rawTx = await getSignedTx(input.id)

  //FIXME make approve function, for check 'from', 'to', 'method'
  //bool status = approve(rawTx)

  const receipt = await signByFeePayerAndExecuteTransaction(rawTx)
  console.log('Receipt:', receipt)
}
