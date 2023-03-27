import {
  Query,
  Logger,
  Controller,
  Get,
  Post,
  Patch,
  Body,
  Delete,
  Param,
  HttpStatus,
  HttpException
} from '@nestjs/common'
import { AggregatorService } from './aggregator.service'
import { ChainService } from '../chain/chain.service'
import { AggregatorDto } from './dto/aggregator.dto'
import { AggregatorQueryDto } from './dto/aggregator-query.dto'
import { AggregatorUpdateDto } from './dto/aggregator-update.dto'
import { IAggregator } from './aggregator.types'

@Controller({
  path: 'aggregator',
  version: '1'
})
export class AggregatorController {
  private readonly logger = new Logger(AggregatorController.name)

  constructor(
    private readonly aggregatorService: AggregatorService,
    private readonly chainService: ChainService
  ) {}

  @Post()
  async create(@Body() aggregatorDto: AggregatorDto) {
    return await this.aggregatorService.create(aggregatorDto)
  }

  @Post('hash')
  async generateHash(@Body() aggregatorDto: AggregatorDto, @Query('verify') verify?: boolean) {
    const aggregator: IAggregator = {
      aggregatorHash: aggregatorDto.aggregatorHash,
      name: aggregatorDto.name,
      heartbeat: aggregatorDto.heartbeat,
      threshold: aggregatorDto.threshold,
      absoluteThreshold: aggregatorDto.absoluteThreshold,
      adapterHash: aggregatorDto.adapterHash
    }

    return await this.aggregatorService.computeAggregatorHash({
      data: aggregator,
      verify: verify
    })
  }

  /**
   * Find all `Aggregator`s based on their `active`ness and assigned `chain`.
   * Used by `Orakl Network Aggregator` during launch of worker.
   *
   * @Query {AggregatorQuerydto}
   */
  @Get()
  async findAll(@Query() query: AggregatorQueryDto) {
    const { chain, active, address } = query

    return await this.aggregatorService.findAll({
      where: {
        chain: { name: chain },
        active,
        address
      }
    })
  }

  /**
   * Find unique `Aggregator` given `aggregatorHash` and `chain`.
   * Used by `Orakl Network Aggregator` to receive metadata about
   * aggregator, its adapter and related data feeds.
   *
   * @Param {string} aggregatorHash
   * @Param {string} chain
   */
  @Get(':hash/:chain')
  async findUnique(@Param('hash') aggregatorHash: string, @Param('chain') chain: string) {
    // chain
    const chainRecord = await this.chainService.findOne({ name: chain })
    if (chainRecord == null) {
      const msg = `Chain [${chain}] not found`
      this.logger.error(msg)
      throw new HttpException(msg, HttpStatus.NOT_FOUND)
    }

    // aggregator
    const aggregatorRecord = await this.aggregatorService.findUnique({
      aggregatorHash_chainId: { aggregatorHash, chainId: chainRecord.id }
    })
    if (aggregatorRecord == null) {
      const msg = `Aggregator [${aggregatorHash}] not found`
      this.logger.error(msg)
      throw new HttpException(msg, HttpStatus.NOT_FOUND)
    }

    try {
      await this.aggregatorService.verifyAggregatorHashOnLoad(
        {
          aggregatorHash_chainId: { aggregatorHash, chainId: chainRecord.id }
        },
        chain
      )
    } catch (e) {
      const msg = `verify Aggregator hash [${aggregatorHash}] failed on load`
      this.logger.error(msg)
      throw new HttpException(msg, HttpStatus.METHOD_NOT_ALLOWED)
    }
    return aggregatorRecord
  }

  @Delete(':id')
  async remove(@Param('id') id: string) {
    return await this.aggregatorService.remove({ id: Number(id) })
  }

  @Patch(':id')
  async update(
    @Param('id') aggregatorHash: string,
    @Body('data') aggregatorUpdateDto: AggregatorUpdateDto
  ) {
    const { id: chainId } = await this.chainService.findOne({ name: aggregatorUpdateDto.chain })
    return await this.aggregatorService.update({
      where: { aggregatorHash_chainId: { aggregatorHash, chainId } },
      active: aggregatorUpdateDto.active
    })
  }
}
