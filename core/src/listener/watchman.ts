import express, { Request, Response } from 'express'
import { Queue } from 'bullmq'
import { Logger } from 'pino'
import { State } from './state'
import { PubSubStop } from './pub-sub-stop'
import { LISTENER_PORT } from '../settings'

export async function watchman({
  listenFn,
  pubsub,
  state,
  logger
}: {
  listenFn
  pubsub: PubSubStop
  state: State
  logger?: Logger
}) {
  const app = express()
  const port =
    /**
     * List all listeners.
     */
    app.get('/all', async (req: Request, res: Response) => {
      try {
        const all = await state.all()
        res.status(200).send(all)
      } catch (e) {
        console.log(e)
        res.status(500).send(e)
      }
    })

  /**
   * List active listeners.
   */
  app.get('/active', async (req: Request, res: Response) => {
    try {
      const active = await state.active()
      res.status(200).send(active)
    } catch (e) {
      console.log(e)
      res.status(500).send(e)
    }
  })

  /**
   * Launch new listener.
   */
  app.get('/start/:id', async (req: Request, res: Response) => {
    const { id } = req.params

    try {
      const listener = await state.add(id)
      listenFn(listener)

      const msg = `Listener with ID=${id} started`
      res.status(200).send(msg)
    } catch (e) {
      console.log(e)
      res.status(500).send(e.message)
    }
  })

  /**
   * Stop a specific listener.
   */
  app.get('/stop/:id', async (req: Request, res: Response) => {
    // const params = req.params
    const { id } = req.params

    try {
      await state.remove(id)
      await pubsub.stop(id)

      const msg = `Listener with ID=${id} stopped`
      res.status(200).send(msg)
    } catch (e) {
      console.log(e)
      res.status(500).send(e.message)
    }
  })

  /**
   * Report on health of listener service.
   */
  app.get('/health', (req: Request, res: Response) => {
    res.send('ok')
  })

  app.listen(LISTENER_PORT, () => {})
}
