export function flattenContract(C) {
  return {
    id: C.id,
    address: C.address,
    allowAllFunctions: C.allowAllFunctions,
    reporter: C.reporter.map((obj) => {
      return obj.address
    }),
    encodedName: C.function.map((obj) => {
      return obj.encodedName
    })
  }
}
