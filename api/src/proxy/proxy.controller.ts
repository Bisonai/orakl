import { Body, Controller, Delete, Get, Param, Patch, Post } from '@nestjs/common'
import { ProxyDto } from './dto/proxy'
import { ProxyService } from './proxy.service'

@Controller({
  path: 'proxy',
  version: '1'
})
export class ProxyController {
  constructor(private readonly proxyService: ProxyService) {}

  @Post()
  async create(@Body() proxyDto: ProxyDto) {
    return await this.proxyService.create(proxyDto)
  }

  @Get()
  async findAll() {
    return await this.proxyService.findAll({ orderBy: { id: 'asc' } })
  }

  @Get(':id')
  async findOne(@Param('id') id: string) {
    return await this.proxyService.findOne({ id: Number(id) })
  }

  @Patch(':id')
  async update(@Param('id') id: string, @Body() proxyDto: ProxyDto) {
    return await this.proxyService.update({
      where: { id: Number(id) },
      proxyDto
    })
  }

  @Delete(':id')
  async remove(@Param('id') id: string) {
    return await this.proxyService.remove({ id: Number(id) })
  }
}
