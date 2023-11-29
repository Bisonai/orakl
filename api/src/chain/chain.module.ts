import { Module } from '@nestjs/common'
import { PrismaService } from '../prisma.service'
import { ChainController } from './chain.controller'
import { ChainService } from './chain.service'

@Module({
  controllers: [ChainController],
  providers: [ChainService, PrismaService]
})
export class ChainModule {}
