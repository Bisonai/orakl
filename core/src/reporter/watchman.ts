import express, { Request, Response } from 'express'
import { Logger } from 'pino'
import { State } from './state'
import { REPORTER_PORT } from '../settings'

export async function watchman({ state, logger }: { state: State; logger: Logger }) {
  const app = express()

  /**
   * List all reporters.
   */
  app.get('/all', async (req: Request, res: Response) => {
    logger.debug('/all')

    try {
      const all = await state.all()
      res.status(200).send(all)
    } catch (e) {
      logger.error(e)
      res.status(500).send(e)
    }
  })

  /**
   * List active reporters.
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
   * Refresh reporters.
   */
  app.get('/refresh', async (req: Request, res: Response) => {
    logger.debug('/refresh')

    try {
      const active = await state.refresh()
      res.status(200).send(active)
    } catch (e) {
      logger.error(e)
      res.status(500).send(e)
    }
  })

  /**
   * Launch new reporter.
   */
  app.get('/activate/:id', async (req: Request, res: Response) => {
    const { id } = req.params
    logger?.debug(`/activate/${id}`)

    try {
      const reporter = await state.add(id)

      const message = `Reporter with ID=${id} started`
      logger?.debug(message)
      res.status(200).send({ message })
    } catch (e) {
      logger?.error(e.message)
      res.status(500).send({ message: e.message })
    }
  })

  /**
   * Stop a specific reporter.
   */
  app.get('/deactivate/:id', async (req: Request, res: Response) => {
    const { id } = req.params
    logger?.debug(`/deactivate/${id}`)

    try {
      await state.remove(id)

      const message = `Reporter with ID=${id} stopped`
      logger?.debug(message)
      res.status(200).send({ message })
    } catch (e) {
      logger?.error(e.message)
      res.status(500).send({ message: e.message })
    }
  })

  /**
   * Report on health of reporter service.
   */
  app.get('/health', (req: Request, res: Response) => {
    logger.debug('/health')
    res.status(200).send('ok')
  })

  app.listen(REPORTER_PORT)
}
