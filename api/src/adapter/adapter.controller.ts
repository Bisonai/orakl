import { Controller, Get, Post, Body, Param, Delete, Query } from '@nestjs/common'
import { AdapterService } from './adapter.service'
import { AdapterDto } from './dto/adapter.dto'

@Controller({
  path: 'adapter',
  version: '1'
})
export class AdapterController {
  constructor(private readonly adapterService: AdapterService) {}

  @Post()
  async create(@Body() adapterDto: AdapterDto) {
    return await this.adapterService.create(adapterDto)
  }

  @Post('hash')
  async generateHash(@Body() adapterDto: AdapterDto, @Query('verify') verify?: boolean) {
    return await this.adapterService.computeAdapterHash({ data: adapterDto, verify: verify })
  }

  @Get()
  async findAll() {
    return await this.adapterService.findAll({})
  }

  @Get(':id')
  async findOne(@Param('id') id: string) {
    return await this.adapterService.findOne({ id: Number(id) })
  }

  @Delete(':id')
  async remove(@Param('id') id: string) {
    return await this.adapterService.remove({ id: Number(id) })
  }
}
