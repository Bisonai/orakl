import { Controller, Get, Post, Body, Patch, Param } from '@nestjs/common'
import { Transaction as TransactionModel } from '@prisma/client'
import { SignService } from './sign.service'
import { SignDto } from './dto/sign.dto'

@Controller({
  path: 'sign',
  version: '1'
})
export class SignController {
  constructor(private readonly signService: SignService) {}

  @Post()
  async create(@Body() signDto: SignDto): Promise<Number> {
    return this.signService.create(signDto)
  }

  @Get()
  findAll() {
    return this.signService.findAll({})
  }

  @Get(':id')
  findOne(@Param('id') id: string) {
    return this.signService.findOne({ id: Number(id) })
  }

  @Patch(':id')
  async update(@Param('id') id: string, @Body() signDto: SignDto): Promise<TransactionModel> {
    return this.signService.update({ where: { id: Number(id) }, signDto })
  }
}
