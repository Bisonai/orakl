function requestResponseConfig() {
  const maxGasLimit = 2_500_000
  const gasAfterPaymentCalculation = 0
  const feeConfig = {
    fulfillmentFlatFeeKlayPPMTier1: 10_000,
    fulfillmentFlatFeeKlayPPMTier2: 10_000,
    fulfillmentFlatFeeKlayPPMTier3: 10_000,
    fulfillmentFlatFeeKlayPPMTier4: 10_000,
    fulfillmentFlatFeeKlayPPMTier5: 10_000,
    reqsForTier2: 0,
    reqsForTier3: 0,
    reqsForTier4: 0,
    reqsForTier5: 0,
  }

  return {
    maxGasLimit,
    gasAfterPaymentCalculation,
    feeConfig,
  }
}

module.exports = {
  requestResponseConfig,
}
