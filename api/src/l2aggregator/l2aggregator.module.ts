import { Module } from '@nestjs/common'
import { L2aggregatorController } from './l2aggregator.controller'
import { L2aggregatorService } from './l2aggregator.service'
import { ChainService } from '../chain/chain.service'
import { PrismaService } from 'src/prisma.service'

@Module({
  controllers: [L2aggregatorController],
  providers: [L2aggregatorService, ChainService, PrismaService]
})
export class L2aggregatorModule {}
