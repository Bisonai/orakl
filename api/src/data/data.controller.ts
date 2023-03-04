import { Controller, Get, Post, Body, Patch, Param, Delete } from '@nestjs/common'
import { DataService } from './data.service'
import { CreateDatumDto } from './dto/create-datum.dto'

@Controller({
  path: 'data',
  version: '1'
})
export class DataController {
  constructor(private readonly dataService: DataService) {}

  @Post()
  create(@Body() createDatumDto: CreateDatumDto) {
    return this.dataService.create(createDatumDto)
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
