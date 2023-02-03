export function vrfConfig() {
  // FIXME
  const oracle = '0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199'
  // FIXME
  const publicProvingKey = [
    '95162740466861161360090244754314042169116280320223422208903791243647772670481',
    '53113177277038648369733569993581365384831203706597936686768754351087979105423'
  ]
  const keyHash = '0x47ede773ef09e40658e643fe79f8d1a27c0aa6eb7251749b268f829ea49f2024'
  const maxGasLimit = 1_000_000
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
    oracle,
    publicProvingKey,
    maxGasLimit,
    keyHash,
    gasAfterPaymentCalculation,
    feeConfig
  }
}
