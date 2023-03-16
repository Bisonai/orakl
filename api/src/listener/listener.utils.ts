import { Listener } from '@prisma/client'

export function flattenListener(L) {
  return {
    id: L.id,
    address: L.address,
    eventName: L.eventName,
    service: L.service.name,
    chain: L.chain.name
  }
}
