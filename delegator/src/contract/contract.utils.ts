export function flattenContract(C) {
  return {
    id: C.id,
    address: C.address,
    allowAllFunctions: C.allowAllFunctions,
    reporter: C.reporter.address,
    encodedName: C.function.encodedName
  }
}
