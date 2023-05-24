import { Module } from "@nestjs/common";
import { DatabaseModule } from "src/modules/database.module";
import { RedisService } from "./redis.service";
import { RedisRepository } from "./redis.repository";
import { RedisController } from "./redis.controller";

@Module({
  imports: [DatabaseModule],
  controllers: [RedisController],
  providers: [RedisService, RedisRepository],
})
export class RedisModule {}
