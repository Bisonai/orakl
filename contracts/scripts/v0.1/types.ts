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
  paymentConfig: IDirectPaymentConfig
  oracle: IVrfOracle[]
}

export interface IRequestResponseConfig {
  maxGasLimit: number
  gasAfterPaymentCalculation: number
  feeConfig: IFeeConfig
  paymentConfig: IDirectPaymentConfig
  oracle: IRequestResponseOracle[]
}

export interface IAggregatorConfig {
  name: string
  paymentAmount: number
  timeout: number
  validator: string
  minSubmissionCount: number
  maxSubmissionCount: number
  decimals: number
  description: string
  restartDelay: number
}
