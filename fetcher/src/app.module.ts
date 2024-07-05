import { BullModule } from '@nestjs/bullmq'
import { Module } from '@nestjs/common'
import { ConfigModule, ConfigService } from '@nestjs/config'
import { AppController } from './app.controller'
import { AppService } from './app.service'
import { JobModule } from './job/job.module'

@Module({
  imports: [
    JobModule,
    ConfigModule.forRoot({
      isGlobal: true,
    }),
    BullModule,
  ],
  controllers: [AppController],
  providers: [AppService, ConfigService],
})
export class AppModule {}
