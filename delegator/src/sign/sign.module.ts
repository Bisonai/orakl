import { Module } from '@nestjs/common'
import { ReporterModule } from '..//reporter/reporter.module'
import { ContractModule } from '../contract/contract.module'
import { FunctionModule } from '../function/function.module'
import { OrganizationModule } from '../organization/organization.module'
import { PrismaService } from '../prisma.service'
import { SignController } from './sign.controller'
import { SignService } from './sign.service'

@Module({
  imports: [OrganizationModule, ContractModule, FunctionModule, ReporterModule],
  controllers: [SignController],
  providers: [SignService, PrismaService]
})
export class SignModule {}
