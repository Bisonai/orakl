export function requestResponseConfig() {
  const maxGasLimit = 2_800_000
  const gasAfterPaymentCalculation = 1_000
  const feeConfig = {
    fulfillmentFlatFeeKlayPPMTier1: 0,
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
