import { Controller, Get, Post, Body, Patch, Param, Delete } from '@nestjs/common'
import { VrfService } from './vrf.service'
import { CreateVrfKeyDto } from './dto/create-vrf-key.dto'
import { UpdateVrfKeyDto } from './dto/update-vrf-key.dto'

@Controller({
  path: 'vrf',
  version: '1'
})
export class VrfController {
  constructor(private readonly vrfService: VrfService) {}

  @Post()
  async create(@Body() createVrfKeyDto: CreateVrfKeyDto) {
    return this.vrfService.create(createVrfKeyDto)
  }

  @Get()
  async findAll() {
    return this.vrfService.findAll({})
  }

  @Get(':id')
  async findOne(@Param('id') id: string) {
    return await this.vrfService.findOne({ id: Number(id) })
  }

  @Patch(':id')
  async update(@Param('id') id: string, @Body() updateVrfKeyDto: UpdateVrfKeyDto) {
    return await this.vrfService.update({
      where: { id: Number(id) },
      updateVrfKeyDto
    })
  }

  @Delete(':id')
  async remove(@Param('id') id: string) {
    return await this.vrfService.remove({ id: Number(id) })
  }
}
