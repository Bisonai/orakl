import { Module } from '@nestjs/common'
import { PrismaService } from '../prisma.service'
import { DataController } from './data.controller'
import { DataService } from './data.service'

@Module({
  controllers: [DataController],
  providers: [DataService, PrismaService]
})
export class DataModule {}
