export function flattenReporter(R) {
  return {
    id: R.id,
    address: R.address,
    organization: R.organization.name,
    contract: R.contract.map((obj) => {
      return obj.address
    })
  }
}
