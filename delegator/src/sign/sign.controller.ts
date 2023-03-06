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
  create(@Body() signDto: SignDto): Promise<TransactionModel> {
    return this.signService.create(signDto)
  }

  @Get()
  findAll() {
    return this.signService.findAll({})
  }

  @Get(':id')
  findOne(@Param('id') id: string) {
    return this.signService.findOne(+id)
  }

  // @Patch(':id')
  // update(@Param('id') id: string, @Body() updateSignDto: UpdateSignDto) {
  //   return this.signService.update(+id, updateSignDto)
  // }
}
