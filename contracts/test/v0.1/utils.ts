export function parseKlay(amount: number) {
  return ethers.utils.parseUnits(amount.toString(), 18)
}
