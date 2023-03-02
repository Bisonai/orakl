import { Controller, Get, Post, Body, Patch, Param, Delete } from '@nestjs/common'
import { Chain as ChainModel } from '@prisma/client'
import { ChainService } from './chain.service'
import { CreateChainDto } from './dto/create-chain.dto'

@Controller('chain')
export class ChainController {
  constructor(private readonly chainService: ChainService) {}

  @Post()
  async create(@Body() createChainDto: CreateChainDto): Promise<ChainModel> {
    return this.chainService.create(createChainDto)
  }

  @Get()
  findAll() {
    return this.chainService.findAll({})
  }

  @Get(':id')
  findOne(@Param('id') id: string) {
    return this.chainService.findOne({ id: Number(id) })
  }

  @Patch(':id')
  async update(@Param('id') id: string, @Body() chainData: { name: string }): Promise<ChainModel> {
    const { name } = chainData
    return this.chainService.update({
      where: { id: Number(id) },
      data: { name }
    })
  }

  @Delete(':id')
  async remove(@Param('id') id: string): Promise<ChainModel> {
    return this.chainService.remove({ id: Number(id) })
  }
}
