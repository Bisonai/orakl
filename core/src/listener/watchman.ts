import express, { Request, Response } from 'express'
import { Queue } from 'bullmq'
import { Logger } from 'pino'
import { getListener } from './api'

const LISTENER_PORT = 4000

export async function watchman({
  redisClient,
  listenFn,
  service,
  chain,
  logger
}: {
  redisClient
  listenFn
  service: string
  chain: string
  logger?: Logger
}) {
  const publisher = redisClient.duplicate()
  await publisher.connect()

  const app = express()
  const port = LISTENER_PORT

  /**
   * List active listeners.
   */
  app.get('/active', (req: Request, res: Response) => {
    res.send('active')
  })

  /**
   * Launch new listener.
   */
  app.get('/start/:id', async (req: Request, res: Response) => {
    const { id } = req.params

    // 1. fetch data from Orakl Network API
    // call start
    try {
      const listener = await getListener({ id, logger })
      if (listener == null) {
        // TODO
        console.log('listener.length == null')
      } else if (listener.service != service || listener.chain != chain) {
        // TODO
        console.log('(listener.service != service || listener.chain != chain)')
      } else {
        listenFn(listener)
      }
    } catch (e) {
      console.log(e)
    }

    res.send('start')
  })

  /**
   * Stop a specific listener.
   */
  app.get('/stop/:id', async (req: Request, res: Response) => {
    const params = req.params
    const channelName = `listener:stop:${params.id}`

    // TODO check on channel name
    await publisher.publish(channelName, 'stop') // FIXME
    res.send('stop')
  })

  /**
   * Report on health of listener service.
   */
  app.get('/health', (req: Request, res: Response) => {
    res.send('ok')
  })

  app.listen(port, () => {})
}
