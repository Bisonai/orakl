import { Module } from '@nestjs/common'
import { PrismaService } from '../prisma.service'
import { ContractController } from './contract.controller'
import { ContractService } from './contract.service'

@Module({
  controllers: [ContractController],
  providers: [ContractService, PrismaService]
})
export class ContractModule {}
