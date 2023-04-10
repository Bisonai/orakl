import { Controller, Get } from '@nestjs/common'
import { AppService } from './app.service'

@Controller({ version: '1' })
export class AppController {
  constructor(private readonly appService: AppService) {}

  @Get()
  root() {
    return this.appService.root()
  }

  @Get('health')
  health() {
    return this.appService.health()
  }
}
