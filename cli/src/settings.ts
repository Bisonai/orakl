export const ORAKL_NETWORK_API_URL =
  process.env.ORAKL_NETWORK_API_URL || 'http://localhost:3000/api/v1'
export const ORAKL_NETWORK_FETCHER_URL =
  process.env.ORAKL_NETWORK_FETCHER_URL || 'http://localhost:3001/api/v1'
export const ORAKL_NETWORK_DELEGATOR_URL =
  process.env.ORAKL_NETWORK_DELEGATOR_URL || 'http://localhost:3002/api/v1'

export const LISTENER_SERVICE_HOST = process.env.LISTENER_SERVICE_HOST || 'http://localhost'
export const LISTENER_SERVICE_PORT = process.env.LISTENER_SERVICE_PORT || 4000

export const WORKER_SERVICE_HOST = process.env.WORKER_SERVICE_HOST || 'http://localhost'
export const WORKER_SERVICE_PORT = process.env.WORKER_SERVICE_PORT || 5001

export const REPORTER_SERVICE_HOST = process.env.REPORTER_SERVICE_HOST || 'http://localhost'
export const REPORTER_SERVICE_PORT = process.env.REPORTER_SERVICE_PORT || 6000
