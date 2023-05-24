import { Module } from '@nestjs/common'
import { DatabaseModule } from "src/modules/database.module";
import { AccountsService } from "./accounts.service";
import { AccountsController } from "./accounts.controller";
import { AccountBalanceRepository } from "./accounts.repository";
import { CommonConfigService } from "src/common/common.config";
import { MonitorConfigService } from 'src/monitor.config/monitor.config.service';
import { MonitorConfigRepository } from 'src/monitor.config/monitor.config.repository';


@Module({
  imports: [DatabaseModule],
  controllers: [AccountsController],
  providers: [AccountsService, AccountBalanceRepository, CommonConfigService, MonitorConfigService, MonitorConfigRepository],
})
export class AccountsModule {}
