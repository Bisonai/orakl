import { Module } from "@nestjs/common";
import { DatabaseModule } from "src/modules/database.module";
import { RedisService } from "./redis.service";
import { RedisRepository } from "./redis.repository";
import { RedisController } from "./redis.controller";
import { AuthModule } from "src/auth/auth.module";
import { JwtService } from "@nestjs/jwt";

@Module({
  imports: [DatabaseModule, AuthModule],
  controllers: [RedisController],
  providers: [RedisService, RedisRepository, JwtService],
})
export class RedisModule {}
