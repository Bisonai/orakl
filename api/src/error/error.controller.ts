import { Body, Controller, Get, Param, Post } from '@nestjs/common'
import { ErrorDto } from './dto/error.dto'
import { ErrorService } from './error.service'

@Controller({
  path: 'error',
  version: '1'
})
export class ErrorController {
  constructor(private readonly errorService: ErrorService) {}

  @Post()
  async create(@Body() errorDto: ErrorDto) {
    return await this.errorService.create(errorDto)
  }

  @Get()
  async findAll() {
    return await this.errorService.findAll({})
  }

  @Get(':id')
  async findOne(@Param('id') id: string) {
    return await this.errorService.findOne({ id: Number(id) })
  }
}
