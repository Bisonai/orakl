export interface IAggregator {
  aggregatorHash?: string
  name: string
  heartbeat: number
  threshold: number
  absoluteThreshold: number
  adapterHash: string
}
