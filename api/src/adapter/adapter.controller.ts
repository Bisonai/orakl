import { Controller, Get, Post, Body, Param, Delete } from '@nestjs/common'
import { Adapter as AdapterModel } from '@prisma/client'
import { AdapterService } from './adapter.service'
import { AdapterDto } from './dto/adapter.dto'

@Controller({
  path: 'adapter',
  version: '1'
})
export class AdapterController {
  constructor(private readonly adapterService: AdapterService) {}

  @Post()
  create(@Body() adapterDto: AdapterDto): Promise<AdapterModel> {
    return this.adapterService.create(adapterDto)
  }

  @Get()
  findAll() {
    return this.adapterService.findAll({})
  }

  @Get(':id')
  findOne(@Param('id') id: string) {
    return this.adapterService.findOne({ id: Number(id) })
  }

  @Delete(':id')
  async remove(@Param('id') id: string): Promise<AdapterModel> {
    return this.adapterService.remove({ id: Number(id) })
  }
}
