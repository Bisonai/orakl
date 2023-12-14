import { Body, Controller, Delete, Get, Param, Patch, Post } from '@nestjs/common'
import { AggregateService } from './aggregate.service'
import { AggregateDto } from './dto/aggregate.dto'

@Controller({
  path: 'aggregate',
  version: '1'
})
export class AggregateController {
  constructor(private readonly aggregateService: AggregateService) {}

  @Post()
  async create(@Body('data') aggregateDto: AggregateDto) {
    return await this.aggregateService.create(aggregateDto)
  }

  @Get()
  async findAll() {
    return await this.aggregateService.findAll({})
  }

  @Get(':id')
  async findOne(@Param('id') id: string) {
    return await this.aggregateService.findOne({ id: Number(id) })
  }

  @Get(['hash/:hash/latest', ':hash/latest'])
  async findLatest(@Param('hash') aggregatorHash: string) {
    return await this.aggregateService.findLatest({ aggregatorHash })
  }

  @Get('id/:id/latest')
  async findLatestById(@Param('id') id: string) {
    return await this.aggregateService.findLatestByAggregatorId({ aggregatorId: BigInt(id) })
  }

  @Patch(':id')
  async update(@Param('id') id: string, @Body() aggregateDto: AggregateDto) {
    return await this.aggregateService.update({
      where: { id: Number(id) },
      aggregateDto
    })
  }

  @Delete(':id')
  async remove(@Param('id') id: string) {
    return await this.aggregateService.remove({ id: Number(id) })
  }
}
