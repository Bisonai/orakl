import { Controller, Get, Post, Body, Patch, Param, Delete } from '@nestjs/common'
import { MethodService } from './method.service'
import { MethodDto } from './dto/method.dto'

@Controller({ path: 'method', version: '1' })
export class MethodController {
  constructor(private readonly methodService: MethodService) {}

  @Post()
  create(@Body() methodDto: MethodDto) {
    return this.methodService.create(methodDto)
  }

  @Get()
  findAll() {
    return this.methodService.findAll({})
  }

  @Get(':id')
  findOne(@Param('id') id: string) {
    return this.methodService.findOne({ id: Number(id) })
  }

  @Patch(':id')
  update(@Param('id') id: string, @Body() methodDto: MethodDto) {
    return this.methodService.update({ where: { id: Number(id) }, methodDto })
  }

  @Delete(':id')
  remove(@Param('id') id: string) {
    return this.methodService.remove({ id: Number(id) })
  }
}
