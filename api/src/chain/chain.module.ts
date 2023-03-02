import { Module } from '@nestjs/common'
import { ChainService } from './chain.service'
import { ChainController } from './chain.controller'
import { PrismaService } from '../prisma.service'

@Module({
  controllers: [ChainController],
  providers: [ChainService, PrismaService]
})
export class ChainModule {}
