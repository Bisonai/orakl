import { Body, Controller, Delete, Get, Param, Patch, Post } from '@nestjs/common'
import { ChainService } from './chain.service'
import { ChainDto } from './dto/chain.dto'

@Controller({
  path: 'chain',
  version: '1'
})
export class ChainController {
  constructor(private readonly chainService: ChainService) {}

  @Post()
  async create(@Body() chainDto: ChainDto) {
    return await this.chainService.create(chainDto)
  }

  @Get()
  async findAll() {
    return await this.chainService.findAll({})
  }

  @Get(':id')
  async findOne(@Param('id') id: string) {
    return await this.chainService.findOne({ id: Number(id) })
  }

  @Patch(':id')
  async update(@Param('id') id: string, @Body() chainDto: ChainDto) {
    return await this.chainService.update({
      where: { id: Number(id) },
      chainDto
    })
  }

  @Delete(':id')
  async remove(@Param('id') id: string) {
    return await this.chainService.remove({ id: Number(id) })
  }
}
