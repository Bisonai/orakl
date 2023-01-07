import * as http from 'http'

export function healthChack() {
  process.env.NODE_ENV !== 'prod' ||
    http
      .createServer(function (_, res) {
        res.writeHead(200, { 'Content-Type': 'text/plain' })
        res.write('ok')
        res.end()
      })
      .listen(8888)
}
