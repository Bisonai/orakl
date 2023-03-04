import { Controller, Get, Post, Body, Patch, Param, Delete } from '@nestjs/common'
import { DataService } from './data.service'
import { DatumDto } from './dto/datum.dto'

@Controller({
  path: 'data',
  version: '1'
})
export class DataController {
  constructor(private readonly dataService: DataService) {}

  @Post()
  create(@Body() datumDto: DatumDto) {
    return this.dataService.create(datumDto)
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
