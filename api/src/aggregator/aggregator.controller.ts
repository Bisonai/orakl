import { Controller, Get, Post, Body, Param } from '@nestjs/common'
import { Aggregator as AggregatorModel } from '@prisma/client'
import { AggregatorService } from './aggregator.service'
import { AggregatorDto } from './dto/aggregator.dto'
import { AggregatorWhereDto } from './dto/aggregator-where.dto'

@Controller({
  path: 'aggregator',
  version: '1'
})
export class AggregatorController {
  constructor(private readonly aggregatorService: AggregatorService) {}

  @Post()
  create(@Body() aggregatorDto: AggregatorDto): Promise<AggregatorModel> {
    return this.aggregatorService.create(aggregatorDto)
  }

  @Get()
  findAll(@Body() aggregatorDto: AggregatorDto) {
    console.log(aggregatorDto)
    return this.aggregatorService.findAll({ where: { chain: { name: 'baobab' } } })
  }

  @Get(':id')
  findOne(@Param('id') id: string) {
    return this.aggregatorService.findOne({ id: Number(id) })
  }
}
