import { Module } from '@nestjs/common'
import { ConfigService } from '@nestjs/config'
import { AppController } from './app.controller'
import { AppService } from './app.service'
import { SignModule } from './sign/sign.module'

@Module({
  imports: [SignModule],
  controllers: [AppController],
  providers: [AppService, ConfigService]
})
export class AppModule {}
