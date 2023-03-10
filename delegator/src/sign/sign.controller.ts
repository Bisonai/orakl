import {
  Controller,
  Get,
  Post,
  Body,
  Patch,
  Param,
  HttpException,
  HttpStatus,
  Delete
} from '@nestjs/common'
import { SignService } from './sign.service'
import { SignDto } from './dto/sign.dto'

@Controller({
  path: 'sign',
  version: '1'
})
export class SignController {
  constructor(private readonly signService: SignService) {}

  @Post()
  async create(@Body() signDto: SignDto) {
    try {
      return await this.signService.create(signDto)
    } catch (e) {
      throw new HttpException(e.message, HttpStatus.FORBIDDEN)
    }
  }

  @Get()
  async findAll() {
    return await this.signService.findAll({})
  }

  @Get(':id')
  async findOne(@Param('id') id: string) {
    return await this.signService.findOne({ id: Number(id) })
  }

  @Patch(':id')
  async update(@Param('id') id: string, @Body() signDto: SignDto) {
    return this.signService.update({ where: { id: Number(id) }, signDto })
  }

  @Delete(':id')
  async remove(@Param('id') id: string) {
    return await this.signService.remove({ id: Number(id) })
  }
}
