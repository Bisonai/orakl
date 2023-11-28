import { Module } from '@nestjs/common'
import { ChainService } from '../chain/chain.service'
import { PrismaService } from '../prisma.service'
import { AggregatorController } from './aggregator.controller'
import { AggregatorService } from './aggregator.service'

@Module({
  controllers: [AggregatorController],
  providers: [AggregatorService, ChainService, PrismaService]
})
export class AggregatorModule {}
