import { Controller, Get, Body, Param, HttpStatus, HttpException, Logger } from '@nestjs/common'
import { InjectQueue } from '@nestjs/bullmq'
import { Queue } from 'bullmq'
import { extractFeeds } from './job.utils'
import {
  loadAggregator,
  activateAggregator,
  deactivateAggregator,
  loadActiveAggregators
} from './job.api'
import { FETCHER_QUEUE_NAME, FETCH_FREQUENCY, FETCHER_TYPE } from '../settings'

@Controller({
  version: '1'
})
export class JobController {
  private readonly logger = new Logger(JobController.name)

  constructor(@InjectQueue(FETCHER_QUEUE_NAME) private queue: Queue) {}

  async onModuleInit() {
    const chain = process.env.CHAIN
    const activeAggregators = await this.activeFetchers()
    for (const aggregator of activeAggregators) {
      await this.startFetcher({ aggregatorHash: aggregator.aggregatorHash, chain, isInitial: true })
    }
  }

  @Get('active')
  async active() {
    return await this.activeFetchers()
  }

  @Get('start/:aggregator')
  async start(@Param('aggregator') aggregatorHash: string, @Body('chain') chain) {
    return await this.startFetcher({ aggregatorHash, chain })
  }

  @Get('stop/:aggregator')
  async stop(@Param('aggregator') aggregatorHash: string, @Body('chain') chain) {
    return await this.stopFetcher({ aggregatorHash, chain })
  }

  private async activeFetchers() {
    const activeAggregators = await loadActiveAggregators({
      chain: process.env.CHAIN,
      logger: this.logger
    })
    const filteredActiveAggregators = activeAggregators.filter(
      (aggregator) => aggregator.fetcherType == FETCHER_TYPE
    )
    return filteredActiveAggregators
  }

  private async startFetcher({
    aggregatorHash,
    chain,
    isInitial
  }: {
    aggregatorHash: string
    chain: string
    isInitial?: boolean
  }) {
    const aggregator = await loadAggregator({ aggregatorHash, chain, logger: this.logger })

    if (Object.keys(aggregator).length == 0) {
      const msg = `Aggregator [${aggregatorHash}] not found`
      this.logger.error(msg)
      throw new HttpException(msg, HttpStatus.NOT_FOUND)
    }

    if (aggregator.fetcherType != FETCHER_TYPE) {
      const msg = `Aggregator [${aggregatorHash}] has different fetcher type than [${FETCHER_TYPE}]`
      this.logger.error(msg)
      throw new HttpException(msg, HttpStatus.BAD_REQUEST)
    }

    if (aggregator.active && !isInitial) {
      const msg = `Aggregator [${aggregatorHash}] is already active`
      this.logger.error(msg)
      throw new HttpException(msg, HttpStatus.BAD_REQUEST)
    }

    const feeds = extractFeeds(
      aggregator.adapter,
      aggregator.id,
      aggregator.aggregatorHash,
      aggregator.threshold,
      aggregator.absoluteThreshold,
      aggregator.address
    )

    // Launch recurrent data collection
    await this.queue.add(aggregatorHash, feeds, {
      repeat: {
        every: FETCH_FREQUENCY
      },
      removeOnComplete: true,
      removeOnFail: true
    })

    try {
      // TODO log the command to separate table
      const res = await activateAggregator(aggregatorHash, chain)
      this.logger.log(res)
    } catch (e) {
      this.logger.error(e)
    }

    const msg = `Activated [${aggregatorHash}]`
    this.logger.log(msg)
    return msg
  }

  private async stopFetcher({ aggregatorHash, chain }: { aggregatorHash: string; chain: string }) {
    const repeatable = await this.queue.getRepeatableJobs()
    const filtered = repeatable.filter((job) => job.name == aggregatorHash)

    if (filtered.length == 1) {
      const job = filtered[0]

      try {
        await this.queue.removeRepeatableByKey(job.key)

        // TODO log the command to separate table
        const res = await deactivateAggregator(aggregatorHash, chain)
        this.logger.log(res)

        const msg = `Deactivated [${aggregatorHash}]`
        this.logger.log(msg)
        return msg
      } catch (e) {
        this.logger.error(e)
        throw new HttpException(e.message, HttpStatus.INTERNAL_SERVER_ERROR)
      }
    } else if (filtered.length == 0) {
      const msg = `Job [${aggregatorHash}] does not exist`
      this.logger.error(msg)
      throw new HttpException(msg, HttpStatus.NOT_FOUND)
    } else {
      const msg = 'Found more than one job satisfying your criteria'
      this.logger.error(msg)
      throw new HttpException(msg, HttpStatus.INTERNAL_SERVER_ERROR)
    }
  }
}
