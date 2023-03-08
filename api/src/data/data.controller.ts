import { Controller, Get, Post, Body, Param } from '@nestjs/common'
import { DataService } from './data.service'
import { DatumDto } from './dto/datum.dto'

@Controller({
  path: 'data',
  version: '1'
})
export class DataController {
  constructor(private readonly dataService: DataService) {}

  @Post()
  async create(@Body('data') dataDto: DatumDto[]) {
    return await this.dataService.createMany(dataDto)
  }

  @Get()
  findAll() {
    return this.dataService.findAll({})
  }

  @Get(':id')
  findOne(@Param('id') id: string) {
    return this.dataService.findOne({ id: Number(id) })
  }
}
