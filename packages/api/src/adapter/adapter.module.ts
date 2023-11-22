import { Module } from '@nestjs/common'
import { AdapterService } from './adapter.service'
import { AdapterController } from './adapter.controller'
import { PrismaService } from '../prisma.service'

@Module({
  controllers: [AdapterController],
  providers: [AdapterService, PrismaService]
})
export class AdapterModule {}
