import { Module } from '@nestjs/common'
import { L2aggregatorController } from './L2aggregator.controller'
import { L2aggregatorService } from './L2aggregator.service'
import { ChainService } from '../chain/chain.service'
import { PrismaService } from '../prisma.service'

@Module({
  controllers: [L2aggregatorController],
  providers: [L2aggregatorService, ChainService, PrismaService]
})
export class L2aggregatorModule {}
