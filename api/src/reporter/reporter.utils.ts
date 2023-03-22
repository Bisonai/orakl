export function flattenReporter(L) {
  return {
    id: L?.id,
    address: L?.address,
    privateKey: L?.privateKey,
    oracleAddress: L?.oracleAddress,
    service: L?.service.name,
    chain: L?.chain?.name
  }
}
