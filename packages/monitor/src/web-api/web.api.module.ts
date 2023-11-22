import { Module } from '@nestjs/common'
import { DatabaseModule } from "src/modules/database.module";
import { CommonConfigService } from "src/common/common.config";
import { MonitorConfigService } from 'src/monitor.config/monitor.config.service';
import { MonitorConfigRepository } from 'src/monitor.config/monitor.config.repository';
import { AuthModule } from 'src/auth/auth.module';
import { JwtService } from '@nestjs/jwt';
import { WebApiController } from './web.api.controller';
import { WebApiService } from './web.api.service';
import { OraklServiceRepository } from './orakl.service.repository';


@Module({
  imports: [DatabaseModule, ],
  controllers: [WebApiController],
  providers: [CommonConfigService, WebApiService, OraklServiceRepository, CommonConfigService],
})
export class WebApiModule {}
