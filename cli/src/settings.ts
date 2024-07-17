const production = process.env.NODE_ENV == 'production'
const default_api_url = production
  ? 'http://orakl-api.orakl.svc.cluster.local'
  : 'http://localhost:3000/api/v1'

const default_delegator_url = production
  ? 'http://orakl-delegator.orakl.svc.cluster.local'
  : 'http://localhost:3002/api/v1'

const default_listener_host = production
  ? 'http://vrf-listener.orakl.svc.cluster.local'
  : 'http://localhost'
const default_listener_port = production ? '4000' : '4000'

const default_worker_host = production
  ? 'http://vrf-worker.orakl.svc.cluster.local'
  : 'http://localhost'
const default_worker_port = production ? '5000' : '5001'

const default_reporter_host = production
  ? 'http://vrf-reporter.orakl.svc.cluster.local'
  : 'http://localhost'
const default_reporter_port = production ? '6000' : '6000'

export const ORAKL_NETWORK_API_URL = process.env.ORAKL_NETWORK_API_URL || default_api_url
export const ORAKL_NETWORK_DELEGATOR_URL =
  process.env.ORAKL_NETWORK_DELEGATOR_URL || default_delegator_url

export const LISTENER_SERVICE_HOST = process.env.LISTENER_SERVICE_HOST || default_listener_host
export const LISTENER_SERVICE_PORT = process.env.LISTENER_SERVICE_PORT || default_listener_port

export const WORKER_SERVICE_HOST = process.env.WORKER_SERVICE_HOST || default_worker_host
export const WORKER_SERVICE_PORT = process.env.WORKER_SERVICE_PORT || default_worker_port

export const REPORTER_SERVICE_HOST = process.env.REPORTER_SERVICE_HOST || default_reporter_host
export const REPORTER_SERVICE_PORT = process.env.REPORTER_SERVICE_PORT || default_reporter_port
