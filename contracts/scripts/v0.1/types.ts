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

export interface IVrfConfig {
  minimumRequestConfirmations: number
  maxGasLimit: number
  gasAfterPaymentCalculation: number
  feeConfig: IFeeConfig
  paymentConfig: IDirectPaymentConfig
  oracle: IVrfOracle[]
}

export interface IRequestResponseConfig {
  minimumRequestConfirmations: number
  maxGasLimit: number
  gasAfterPaymentCalculation: number
  feeConfig: IFeeConfig
}
