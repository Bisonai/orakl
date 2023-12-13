import { Injectable, OnApplicationShutdown, OnModuleInit } from '@nestjs/common'
import type { RedisClientType } from 'redis'
import { createClient } from 'redis'

@Injectable()
export class RedisService implements OnModuleInit, OnApplicationShutdown {
  private readonly redisClient: RedisClientType

  constructor() {
    const isProduction = process.env.NODE_ENV == 'production'
    const envRedisHost = process.env.REDIS_HOST
    const envRedisPort = process.env.REDIS_PORT

    const redisHost =
      envRedisHost ||
      (isProduction ? 'redis-data-feed-master.redis.svc.cluster.local' : 'localhost')
    const redisPort = envRedisPort || '6379'

    this.redisClient = createClient({
      url: `redis://${redisHost}:${redisPort}`
    })
  }
  async onModuleInit() {
    await this.redisClient.connect()
  }

  async onApplicationShutdown() {
    await this.redisClient.disconnect()
  }

  async set(key: string, value: string): Promise<void> {
    await this.redisClient.set(key, value)
  }

  async get(key: string): Promise<string | null> {
    return this.redisClient.get(key)
  }
}
