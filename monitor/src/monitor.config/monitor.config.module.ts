import { Module } from "@nestjs/common";
import { DatabaseModule } from "src/modules/database.module";
import { MonitorConfigRepository } from "./monitor.config.repository";
import { MonitorConfigService } from "./monitor.config.service";
import { MonitorConfigController } from "./monitor.config.controller";


@Module({
  imports: [DatabaseModule],
  controllers: [MonitorConfigController],
  providers: [MonitorConfigService, MonitorConfigRepository],
})
export class MonitorConfigModule {}
