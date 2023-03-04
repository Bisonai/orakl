import { Module } from '@nestjs/common'
import { AggregatorService } from './aggregator.service'
import { AggregatorController } from './aggregator.controller'
import { PrismaService } from '../prisma.service'

@Module({
  controllers: [AggregatorController],
  providers: [AggregatorService, PrismaService]
})
export class AggregatorModule {}
