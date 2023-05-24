import { Module } from "@nestjs/common";
import { ConfigModule } from "@nestjs/config";
import { RedisConfigService } from "src/common/redis.config";

@Module({
  imports: [ConfigModule],
  providers: [
    RedisConfigService,
    {
      provide: "VRF_REDIS",
      useFactory: async (config: RedisConfigService) => config.vrf,
      inject: [RedisConfigService],
    },
    {
      provide: "REQUEST_RESPONSE_REDIS",
      useFactory: async (config: RedisConfigService) => config.requestResponse,
      inject: [RedisConfigService],
    },
    {
      provide: "AGGREGATOR_REDIS",
      useFactory: async (config: RedisConfigService) => config.aggregator,
      inject: [RedisConfigService],
    },
    {
      provide: "FETCHER_REDIS",
      useFactory: async (config: RedisConfigService) => config.fetcher,
      inject: [RedisConfigService],
    },
  ],
  exports: [
    "VRF_REDIS",
    "REQUEST_RESPONSE_REDIS",
    "AGGREGATOR_REDIS",
    "FETCHER_REDIS",
  ],
})
export class OldRedisModule {}
