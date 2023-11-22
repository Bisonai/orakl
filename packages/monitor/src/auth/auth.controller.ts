import {
    Body,
    Controller,
    Get,
    HttpCode,
    HttpStatus,
    Post,
    Request,
    UseGuards
  } from '@nestjs/common';
  import { AuthService } from './auth.service';
import { ApiProperty } from '@nestjs/swagger';
  
export class PasswordDto {
  @ApiProperty({
    example: '3425343334',
    description: 'password',
  })
  password: string;
  
}

  @Controller('auth')
  export class AuthController {
    constructor(private authService: AuthService) {}
  
    @HttpCode(HttpStatus.OK)
    @Post('login')  
    signIn(@Body() signInDto: PasswordDto) {
      return this.authService.signIn(signInDto.password);
    }
  
  }