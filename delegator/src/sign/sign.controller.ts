import { Body, Controller, Get, HttpException, HttpStatus, Post } from '@nestjs/common'
import { SignDto } from './dto/sign.dto'
import { SignService } from './sign.service'

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

  @Get('initialize')
  initialize() {
    return this.signService.initialize({})
  }
}
