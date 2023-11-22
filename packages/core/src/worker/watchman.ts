import express, { Request, Response } from 'express'
import { Logger } from 'pino'
import { WORKER_PORT } from '../settings'
import { State } from './state'

export async function watchman({ state, logger }: { state: State; logger: Logger }) {
  const app = express()

  /**
   * List all workers.
   */
  app.get('/all', async (req: Request, res: Response) => {
    logger.debug('/all')

    try {
      const all = await state.all()
      res.status(200).send(all)
    } catch (e) {
      logger?.error(e)
      res.status(500).send(e)
    }
  })

  /**
   * List active workers.
   */
  app.get('/active', async (req: Request, res: Response) => {
    logger.debug('/active')

    try {
      const active = await state.active()
      res.status(200).send(active)
    } catch (e) {
      logger.error(e)
      res.status(500).send(e)
    }
  })

  /**
   * Launch new worker.
   */
  app.get('/activate/:aggregatorHash', async (req: Request, res: Response) => {
    const { aggregatorHash } = req.params
    logger.debug(`/activate/${aggregatorHash}`)

    try {
      await state.add(aggregatorHash)

      const message = `Worker with aggregatorHash=${aggregatorHash} started`
      logger.debug(message)
      res.status(200).send({ message })
    } catch (e) {
      logger.error(e.message)
      res.status(500).send({ message: e.message })
    }
  })

  /**
   * Stop a specific worker.
   */
  app.get('/deactivate/:aggregatorHash', async (req: Request, res: Response) => {
    const { aggregatorHash } = req.params
    logger.debug(`/deactivate/${aggregatorHash}`)

    try {
      await state.remove(aggregatorHash)

      const message = `Worker with aggregatorHash=${aggregatorHash} stopped`
      logger.debug(message)
      res.status(200).send({ message })
    } catch (e) {
      logger.error(e.message)
      res.status(500).send({ message: e.message })
    }
  })

  /**
   * Report on health of worker service.
   */
  app.get('/health', (req: Request, res: Response) => {
    logger.debug('/health')
    res.status(200).send('ok')
  })

  return app.listen(WORKER_PORT)
}
