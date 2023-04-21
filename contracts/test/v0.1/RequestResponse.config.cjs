function requestResponseConfig() {
  const maxGasLimit = 2_500_000
  const gasAfterPaymentCalculation = 1_000
  const feeConfig = {
    fulfillmentFlatFeeKlayPPMTier1: 10_000,
    fulfillmentFlatFeeKlayPPMTier2: 0,
    fulfillmentFlatFeeKlayPPMTier3: 0,
    fulfillmentFlatFeeKlayPPMTier4: 0,
    fulfillmentFlatFeeKlayPPMTier5: 0,
    reqsForTier2: 0,
    reqsForTier3: 0,
    reqsForTier4: 0,
    reqsForTier5: 0
  }
  const directFeeConfig = {
    fulfillmentFee: 5_000_000_000_000_000,
    baseFee: 5_000_000_000
  }

  return {
    maxGasLimit,
    gasAfterPaymentCalculation,
    feeConfig,
    directFeeConfig
  }
}

module.exports = {
  requestResponseConfig
}
