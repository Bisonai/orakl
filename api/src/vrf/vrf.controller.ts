import { Body, Controller, Delete, Get, Param, Patch, Post } from '@nestjs/common'
import { CreateVrfKeyDto } from './dto/create-vrf-key.dto'
import { UpdateVrfKeyDto } from './dto/update-vrf-key.dto'
import { VrfService } from './vrf.service'

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
  async findAll(@Body('chain') chain: string) {
    return this.vrfService.findAll({ where: { chain: { name: chain } } })
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
