import { Body, Controller, Delete, Get, Param, Patch, Post } from '@nestjs/common'
import { ServiceDto } from './dto/service.dto'
import { ServiceService } from './service.service'

@Controller({
  path: 'service',
  version: '1'
})
export class ServiceController {
  constructor(private readonly serviceService: ServiceService) {}

  @Post()
  async create(@Body() serviceDto: ServiceDto) {
    return await this.serviceService.create(serviceDto)
  }

  @Get()
  async findAll() {
    return await this.serviceService.findAll({})
  }

  @Get(':id')
  async findOne(@Param('id') id: string) {
    return await this.serviceService.findOne({ id: Number(id) })
  }

  @Patch(':id')
  async update(@Param('id') id: string, @Body() serviceDto: ServiceDto) {
    return await this.serviceService.update({
      where: { id: Number(id) },
      serviceDto
    })
  }

  @Delete(':id')
  async remove(@Param('id') id: string) {
    return await this.serviceService.remove({ id: Number(id) })
  }
}
