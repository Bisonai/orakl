import { Module } from '@nestjs/common'
import { PrismaService } from '../prisma.service'
import { AdapterController } from './adapter.controller'
import { AdapterService } from './adapter.service'

@Module({
  controllers: [AdapterController],
  providers: [AdapterService, PrismaService]
})
export class AdapterModule {}
