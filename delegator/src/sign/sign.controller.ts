import { Controller, Post, Body, HttpException, HttpStatus } from '@nestjs/common'
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
}
