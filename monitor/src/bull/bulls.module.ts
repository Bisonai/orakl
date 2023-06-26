import { Module } from '@nestjs/common'
import { BullsService } from './bulls.service';
import { BullsController } from './bulls.controller';
import { BullModule } from '@nestjs/bullmq';
import { BullsRepository } from './bulls.repository';
import { DatabaseModule } from "src/modules/database.module";
import { RedisService } from "src/redis/redis.service";
import { RedisModule } from "src/redis/redis.module";
import { RedisRepository } from "src/redis/redis.repository";
import { MonitorConfigService } from 'src/monitor.config/monitor.config.service';
import { MonitorConfigRepository } from 'src/monitor.config/monitor.config.repository';
import { AuthModule } from 'src/auth/auth.module';
import { JwtService } from '@nestjs/jwt';

@Module({
  imports: [DatabaseModule, AuthModule],
  controllers: [BullsController],
  providers: [BullsService, BullsRepository, RedisService, RedisRepository, MonitorConfigService, MonitorConfigRepository, JwtService],
})
export class BullsModule {}