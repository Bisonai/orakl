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
    switch (serviceName) {
      case SERVICE.VRF:
        return {
          host: process.env.VRF_REDIS_HOST || redisInfo?.host,
          port: parseInt(process.env.VRF_REDIS_PORT, 10) || redisInfo?.port,
        };
      case SERVICE.REQUEST_RESPONSE:
        return {
          host: process.env.REQUEST_RESPONSE_REDIS_HOST || redisInfo?.host,
          port:
            parseInt(process.env.REQUEST_RESPONSE_REDIS_PORT, 10) ||
            redisInfo?.port,
        };
      case SERVICE.AGGREGATOR:
        return {
          host: process.env.AGGREGATOR_REDIS_HOST || redisInfo?.host,
          port:
            parseInt(process.env.AGGREGATOR_REDIS_PORT, 10) || redisInfo?.port,
        };
      case SERVICE.FETCHER:
        return {
          host: process.env.FETCHER_REDIS_HOST || redisInfo?.host,
          port: parseInt(process.env.FETCHER_REDIS_PORT, 10) || redisInfo?.port,
        };
      default:
        return { host: "localhost", port: 6379 };
        break;
    }
  }
}
