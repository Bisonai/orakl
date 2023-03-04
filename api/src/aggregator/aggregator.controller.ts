import { Controller, Get, Post, Body, Param } from '@nestjs/common'
import { Aggregator as AggregatorModel } from '@prisma/client'
import { AggregatorService } from './aggregator.service'
import { CreateAggregatorDto } from './dto/create-aggregator.dto'

@Controller({
  path: 'aggregator',
  version: '1'
})
export class AggregatorController {
  constructor(private readonly aggregatorService: AggregatorService) {}

  @Post()
  create(@Body() createAggregatorDto: CreateAggregatorDto): Promise<AggregatorModel> {
    return this.aggregatorService.create(createAggregatorDto)
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
