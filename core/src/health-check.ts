import * as http from 'http'
import { HEALTH_CHECK_PORT, NODE_ENV } from './settings'

export function launchHealthCheck() {
  if (NODE_ENV == 'production') {
    http
      .createServer(function (_, res) {
        res.writeHead(200, { 'Content-Type': 'text/plain' })
        res.write('ok')
        res.end()
      })
      .listen(HEALTH_CHECK_PORT)
  }
}
