import { Module } from '@nestjs/common'
import { ServiceService } from './service.service'
import { ServiceController } from './service.controller'
import { PrismaService } from '../prisma.service'

@Module({
  controllers: [ServiceController],
  providers: [ServiceService, PrismaService]
})
export class ServiceModule {}
