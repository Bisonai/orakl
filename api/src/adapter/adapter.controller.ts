import { Controller, Get, Post, Body, Param } from '@nestjs/common'
import { Adapter as AdapterModel } from '@prisma/client'
import { AdapterService } from './adapter.service'
import { CreateAdapterDto } from './dto/create-adapter.dto'

@Controller({
  path: 'adapter',
  version: '1'
})
export class AdapterController {
  constructor(private readonly adapterService: AdapterService) {}

  @Post()
  create(@Body() createAdapterDto: CreateAdapterDto): Promise<AdapterModel> {
    return this.adapterService.create(createAdapterDto)
  }

  @Get()
  findAll() {
    return this.adapterService.findAll({})
  }

  @Get(':id')
  findOne(@Param('id') id: string) {
    return this.adapterService.findOne({ id: Number(id) })
  }
}
