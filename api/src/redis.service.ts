// redis.service.ts

import { Injectable, OnApplicationShutdown, OnModuleInit } from '@nestjs/common'
import type { RedisClientType } from 'redis'
import { createClient } from 'redis'

@Injectable()
export class RedisService implements OnModuleInit, OnApplicationShutdown {
  private readonly redisClient: RedisClientType

  constructor() {
    this.redisClient = createClient({
      url: `redis://${process.env.REDIS_HOST}:${process.env.REDIS_PORT}`
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
