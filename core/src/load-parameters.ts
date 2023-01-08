import * as dotenv from 'dotenv'
dotenv.config()

function cantBeEmptyString(s) {
  if (s.length == 0) {
    throw Error()
  } else {
    return s
  }
}

export const NODE_ENV = process.env.NODE_ENV

export const PROVIDER_URL = process.env.PROVIDER

export const REDIS_HOST = process.env.REDIS_HOST || 'localhost'
export const REDIS_PORT = Number(process.env.REDIS_PORT) || 6379

export const LOCAL_AGGREGATOR = process.env.LOCAL_AGGREGATOR

export const PROVIDER = process.env.PROVIDER

// FIXME allow either private key or mnemonic
export const MNEMONIC = process.env.MNEMONIC
export const PRIVATE_KEY = process.env.PRIVATE_KEY

export const HEALTH_CHECK_PORT = process.env.HEALTH_CHECK_PORT
