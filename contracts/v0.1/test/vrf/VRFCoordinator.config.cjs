function vrfConfig() {
  const maxGasLimit = 2_500_000
  const gasAfterPaymentCalculation = 50_000
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

  // The following settings were generate using `yarn cli vrf keygen`
  const sk = 'b368e407363d7435903b1511025bc8345b76aa4fcfe7ab36fb8a71349e1fe95a'
  const pk =
    '045012f4b244b7875e34ac2af0856e463c2c5c94fe754e90a844798047aa32ae34a546a783546ca30b6044ce9612c2d458815249a5f460eb3ce878adf1dc55dec7'
  const pkX = '36218519966043180833110848345962110858668389778776319719451647092795737550388'
  const pkY = '74756455443057291531071945062373943175438004978429662040378909820867954728647'
  const publicProvingKey = [pkX, pkY]
  const keyHash = '0x1833807c931ca83e42ada8a2730626cdd00871e3013927a2b89f94e82a6844dd'

  return {
    maxGasLimit,
    gasAfterPaymentCalculation,
    feeConfig,
    sk,
    pk,
    pkX,
    pkY,
    publicProvingKey,
    keyHash,
  }
}

module.exports = {
  vrfConfig,
}
