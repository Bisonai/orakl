export function flattenFunction(F) {
  return {
    id: F.id,
    name: F.name,
    encodedName: F.encodedName,
    address: F.contract.address
  }
}
