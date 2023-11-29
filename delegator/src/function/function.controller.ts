import { Body, Controller, Delete, Get, Param, Patch, Post } from '@nestjs/common'
import { FunctionDto } from './dto/function.dto'
import { FunctionService } from './function.service'

@Controller({ path: 'function', version: '1' })
export class FunctionController {
  constructor(private readonly functionService: FunctionService) {}

  @Post()
  create(@Body() functionDto: FunctionDto) {
    return this.functionService.create(functionDto)
  }

  @Get()
  findAll() {
    return this.functionService.findAll({})
  }

  @Get(':id')
  findOne(@Param('id') id: string) {
    return this.functionService.findOne({ id: Number(id) })
  }

  @Patch(':id')
  update(@Param('id') id: string, @Body() functionDto: FunctionDto) {
    return this.functionService.update({ where: { id: Number(id) }, functionDto })
  }

  @Delete(':id')
  remove(@Param('id') id: string) {
    return this.functionService.remove({ id: Number(id) })
  }
}
