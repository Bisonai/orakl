import * as http from 'http'
import { NODE_ENV, HEALTH_CHECK_PORT } from './load-parameters'

export function healthCheck() {
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
