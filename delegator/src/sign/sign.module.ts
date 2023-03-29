import { Module } from '@nestjs/common'
import { SignService } from './sign.service'
import { PrismaService } from '../prisma.service'
import { SignController } from './sign.controller'
import { OrganizationModule } from 'src/organization/organization.module'
import { ContractModule } from 'src/contract/contract.module'
import { FunctionModule } from 'src/function/function.module'
import { ReporterModule } from 'src/reporter/reporter.module'

@Module({
  imports: [OrganizationModule, ContractModule, FunctionModule, ReporterModule],
  controllers: [SignController],
  providers: [SignService, PrismaService]
})
export class SignModule {}
