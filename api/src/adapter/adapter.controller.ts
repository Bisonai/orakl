import { Controller, Get, Post, Body, Patch, Param, Delete } from '@nestjs/common'
import { Adapter as AdapterModel } from '@prisma/client'
import { AdapterService } from './adapter.service'
import { CreateAdapterDto } from './dto/create-adapter.dto'
import { UpdateAdapterDto } from './dto/update-adapter.dto'

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

  @Patch(':id')
  update(@Param('id') id: string, @Body() updateAdapterDto: UpdateAdapterDto) {
    return this.adapterService.update(+id, updateAdapterDto)
  }

  @Delete(':id')
  remove(@Param('id') id: string) {
    return this.adapterService.remove(+id)
  }
}
