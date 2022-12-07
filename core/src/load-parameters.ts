import * as dotenv from 'dotenv'
dotenv.config()

export const PROVIDER_URL = process.env.PROVIDER
export const LISTENERS_PATH = process.env.LISTENERS // FIXME raise error when file does not exist

export const VRF_SK = process.env.VRF_SK
export const VRF_PK = process.env.VRF_PK

export const REDIS_HOST = process.env.REDIS_HOST || 'localhost'
export const REDIS_PORT = Number(process.env.REDIS_PORT) || 6379

export const LOCAL_AGGREGATOR = process.env.LOCAL_AGGREGATOR

export const PROVIDER = process.env.PROVIDER

// FIXME allow either private key or mnemonic
export const MNEMONIC = process.env.MNEMONIC
export const PRIVATE_KEY = process.env.PRIVATE_KEY
