import { Module } from '@nestjs/common'
import { ConfigService } from '@nestjs/config'
import { AppController } from './app.controller'
import { AppService } from './app.service'
import { ContractModule } from './contract/contract.module'
import { FunctionModule } from './function/function.module'
import { OrganizationModule } from './organization/organization.module'
import { ReporterModule } from './reporter/reporter.module'
import { SignModule } from './sign/sign.module'

@Module({
  imports: [SignModule, OrganizationModule, ContractModule, FunctionModule, ReporterModule],
  controllers: [AppController],
  providers: [AppService, ConfigService]
})
export class AppModule {}
