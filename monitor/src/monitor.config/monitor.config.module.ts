import { Module } from "@nestjs/common";
import { DatabaseModule } from "src/modules/database.module";
import { MonitorConfigRepository } from "./monitor.config.repository";
import { MonitorConfigService } from "./monitor.config.service";
import { MonitorConfigController } from "./monitor.config.controller";
import { AuthModule } from "src/auth/auth.module";
import { JwtService } from "@nestjs/jwt";


@Module({
  imports: [DatabaseModule, AuthModule],
  controllers: [MonitorConfigController],
  providers: [MonitorConfigService, MonitorConfigRepository, JwtService],
})
export class MonitorConfigModule {}
