import { Module } from '@nestjs/common';
import { ConfigModule, ConfigService } from '@nestjs/config';
import { AppService } from './app.service';
import { AppController } from './app.controller';
import { AccountsModule } from './accounts/accounts.module';

import { AccountBalanceRepository } from './accounts/accounts.repository';
import { DatabaseModule } from "./modules/database.module";
import { commonConfig, databaseConfig } from "./common/configuration";
import { BullsModule } from "./bull/bulls.module";
import { RedisModule } from "./redis/redis.module";
import { ScheduleModule } from "@nestjs/schedule";
import { AccountsService } from "./accounts/accounts.service";
import { CommonConfigService } from "./common/common.config";
import { MonitorConfigModule } from "./monitor.config/monitor.config.module";
import { MonitorConfigService } from "./monitor.config/monitor.config.service";
import { MonitorConfigRepository } from "./monitor.config/monitor.config.repository";

@Module({
  imports: [
    ConfigModule.forRoot({
      isGlobal: true,
      load: [databaseConfig, commonConfig],
    }),
    ScheduleModule.forRoot(),
    RedisModule,
    AccountsModule,
    BullsModule,
    DatabaseModule,
  ],
  controllers: [AppController],
  providers: [
    AppService,
    AccountBalanceRepository,
    AccountsService,
    CommonConfigService,
    MonitorConfigService,
    MonitorConfigRepository,
  ],
})
export class AppModule {}
