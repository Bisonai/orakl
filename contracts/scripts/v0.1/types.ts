interface IDirectPaymentConfig {
  fulfillmentFee: string
  baseFee: string
}

interface IFeeConfig {
  fulfillmentFlatFeeKlayPPMTier1: number
  fulfillmentFlatFeeKlayPPMTier2: number
  fulfillmentFlatFeeKlayPPMTier3: number
  fulfillmentFlatFeeKlayPPMTier4: number
  fulfillmentFlatFeeKlayPPMTier5: number
  reqsForTier2: number
  reqsForTier3: number
  reqsForTier4: number
  reqsForTier5: number
}

interface IVrfOracle {
  address: string
  publicProvingKey: [string, string]
}

interface IRequestResponseOracle {
  address: string
}

export interface IVrfConfig {
  maxGasLimit: number
  gasAfterPaymentCalculation: number
  feeConfig: IFeeConfig
  directPaymentConfig: IDirectPaymentConfig
  oracle: IVrfOracle[]
  minBalance: string
}

export interface IRequestResponseConfig {
  maxGasLimit: number
  gasAfterPaymentCalculation: number
  feeConfig: IFeeConfig
  directPaymentConfig: IDirectPaymentConfig
  oracle: IRequestResponseOracle[]
  minBalance: string
}

interface IAggregatorDeployConfig {
  name: string
  paymentAmount: number
  timeout: number
  validator: string
  decimals: number
  description: string
  depositAmount?: number
}

interface IAggregatorChangeOraclesConfig {
  removed: string[]
  added: string[]
  addedAdmins: string[]
  minSubmissionCount: number
  maxSubmissionCount: number
  restartDelay: number
  aggregatorAddress?: string
}

export interface IAggregatorConfig {
  deploy?: IAggregatorDeployConfig
  changeOracles?: IAggregatorChangeOraclesConfig
}
