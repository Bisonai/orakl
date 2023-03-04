import { Controller, Get, Post, Body, Param } from '@nestjs/common'
import { Aggregator as AggregatorModel } from '@prisma/client'
import { AggregatorService } from './aggregator.service'
import { AggregatorDto } from './dto/aggregator.dto'

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
  findAll() {
    return this.aggregatorService.findAll({})
  }

  @Get(':id')
  findOne(@Param('id') id: string) {
    return this.aggregatorService.findOne({ id: Number(id) })
  }
}
