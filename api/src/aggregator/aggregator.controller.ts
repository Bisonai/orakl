import { Controller, Get, Post, Patch, Body, Delete, Param } from '@nestjs/common'
import { AggregatorService } from './aggregator.service'
import { ChainService } from '../chain/chain.service'
import { AggregatorDto } from './dto/aggregator.dto'
import { AggregatorWhereDto } from './dto/aggregator-where.dto'
import { AggregatorUpdateDto } from './dto/aggregator-update.dto'

@Controller({
  path: 'aggregator',
  version: '1'
})
export class AggregatorController {
  constructor(
    private readonly aggregatorService: AggregatorService,
    private readonly chainService: ChainService
  ) {}

  @Post()
  async create(@Body() aggregatorDto: AggregatorDto) {
    return await this.aggregatorService.create(aggregatorDto)
  }

  @Get()
  findAll(@Body() whereDto: AggregatorWhereDto) {
    return this.aggregatorService.findAll({
      where: {
        chain: { name: whereDto.chain },
        active: whereDto.active
      }
    })
  }

  @Get(':id')
  async findOne(@Param('id') aggregatorHash: string, @Body('chain') chain) {
    const { id: chainId } = await this.chainService.findOne({ name: chain })
    return await this.aggregatorService.findOne({
      aggregatorHash_chainId: { aggregatorHash, chainId }
    })
  }

  @Delete(':id')
  async remove(@Param('id') id: string) {
    return this.aggregatorService.remove({ id: Number(id) })
  }

  @Patch(':id')
  async update(
    @Param('id') aggregatorHash: string,
    @Body('data') aggregatorUpdateDto: AggregatorUpdateDto
  ) {
    const { id: chainId } = await this.chainService.findOne({ name: aggregatorUpdateDto.chain })
    return this.aggregatorService.update({
      where: { aggregatorHash_chainId: { aggregatorHash, chainId } },
      active: aggregatorUpdateDto.active
    })
  }
}
