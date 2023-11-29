import { Module } from '@nestjs/common'
import { PrismaService } from '../prisma.service'
import { ServiceController } from './service.controller'
import { ServiceService } from './service.service'

@Module({
  controllers: [ServiceController],
  providers: [ServiceService, PrismaService]
})
export class ServiceModule {}
