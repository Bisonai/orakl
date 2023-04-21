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

  return {
    maxGasLimit,
    gasAfterPaymentCalculation,
    feeConfig
  }
}

module.exports = {
  requestResponseConfig
}
