const { ethers } = require('hardhat')
const dotenv = require('dotenv')
dotenv.config()

// Input configuration to create Prepayment Account
const START_TIME = Math.round(new Date().getTime() / 1000) - 60 * 60 // Start time in seconds
const PERIOD = 60 * 60 * 24 * 7 // Duration in seconds
const REQUEST_NUMBER = 10 // Number of requests
const OWNER_ADDRESS = '0x30E30C3B6313FF232E93593b883fC8A8AF8BB627' // Account Owner Address

const FEE_RATIO = 10000 // 100%
const SUBSCRIPTION_PRICE = ethers.utils.parseEther('1') // Subscription price in Klay

const ACC_ID = 1

const PROVIDER_URL = process.env.PROVIDER || 'http://127.0.0.1:8545'
const MNEMONIC = process.env.MNEMONIC

module.exports = {
  START_TIME,
  PERIOD,
  REQUEST_NUMBER,
  OWNER_ADDRESS,
  FEE_RATIO,
  SUBSCRIPTION_PRICE,
  ACC_ID,
  PROVIDER_URL,
  MNEMONIC
}
