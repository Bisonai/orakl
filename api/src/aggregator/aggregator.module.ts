import { Module } from '@nestjs/common'
import { AggregatorService } from './aggregator.service'
import { ChainService } from '../chain/chain.service'
import { AggregatorController } from './aggregator.controller'
import { PrismaService } from '../prisma.service'

@Module({
  controllers: [AggregatorController],
  providers: [AggregatorService, ChainService, PrismaService]
})
export class AggregatorModule {}
