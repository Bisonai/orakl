import { Injectable } from "@nestjs/common";
import { RedisRepository } from "./redis.repository";
import { SERVICE } from "src/common/types";

@Injectable()
export class RedisService {
  constructor(private readonly redisRepository: RedisRepository) {}

  async registerRedis(serviceName: SERVICE, host, port) {
    return await this.redisRepository.create(serviceName, host, port);
  }

  async findRedis(serviceName) {
    const redisInfo = await this.redisRepository.findOne(serviceName);
    const host = redisInfo?.host || "localhost";
    const port = redisInfo?.port || 5432;
    return { host, port };
  }
}
