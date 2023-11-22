import { Controller, Get, Post, Body, Patch, Param, Delete } from '@nestjs/common'
import { OrganizationService } from './organization.service'
import { OrganizationDto } from './dto/organization.dto'

@Controller({
  path: 'organization',
  version: '1'
})
export class OrganizationController {
  constructor(private readonly organizationService: OrganizationService) {}

  @Post()
  async create(@Body() organizationDto: OrganizationDto) {
    return await this.organizationService.create(organizationDto)
  }

  @Get()
  async findAll() {
    return await this.organizationService.findAll({})
  }

  @Get(':id')
  async findOne(@Param('id') id: string) {
    return await this.organizationService.findOne({ id: Number(id) })
  }

  @Patch(':id')
  async update(@Param('id') id: string, @Body() organizationDto: OrganizationDto) {
    return await this.organizationService.update({
      where: { id: Number(id) },
      organizationDto
    })
  }

  @Delete(':id')
  async remove(@Param('id') id: string) {
    return await this.organizationService.remove({ id: Number(id) })
  }
}
