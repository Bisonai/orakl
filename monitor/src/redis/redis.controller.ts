import { Controller, HttpCode, HttpStatus, Param, Post, Put, UseGuards } from "@nestjs/common";
import { ApiOperation, ApiTags } from "@nestjs/swagger";
import { RedisService } from "./redis.service";
import { SERVICE } from "src/common/types";
import { AuthGuard } from "src/auth/auth.guard";

@Controller("redis")
@ApiTags("redis")
export class RedisController {
  constructor(private readonly redisService: RedisService) {}

  @Post("/register/:service_name/:host/:port")
  @UseGuards(AuthGuard)
  @ApiOperation({ operationId: "registerRedis" })
  @HttpCode(HttpStatus.OK)
  async createRedis(
    @Param("service_name") serviceName: SERVICE,
    @Param("host") host: string,
    @Param("port") port: number
  ) {
    return await this.redisService.registerRedis(serviceName, host, port);
  }
}
