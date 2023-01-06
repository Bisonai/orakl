import { HealthStatus } from './types'

/**
 * A simple liveness check
 * @returns HealthStatus
 */
export function healthCheck(): HealthStatus {
  /* you can add health check logic if you need */
  return {
    status: 'success'
  }
}
