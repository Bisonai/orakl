import { Module } from '@nestjs/common'
import { DataService } from './data.service'
import { PrismaService } from '../prisma.service'
import { DataController } from './data.controller'

@Module({
  controllers: [DataController],
  providers: [DataService, PrismaService]
})
export class DataModule {}
