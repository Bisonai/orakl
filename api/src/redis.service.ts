// redis.service.ts

import { Injectable } from '@nestjs/common'
import type { RedisClientType } from 'redis'
import { createClient } from 'redis'

@Injectable()
export class RedisService {
  private readonly redisClient: RedisClientType;

  constructor() {
    this.redisClient = createClient({
        url: `redis://${process.env.REDIS_HOST}:${process.env.REDIS_PORT}`
      });
  }

  async set(key: string, value: string): Promise<void> {
    // Set a value in Redis
    await this.redisClient.set(key, value);
  }

  async get(key: string): Promise<string | null> {
    // Get a value from Redis
    return this.redisClient.get(key);
  }
}
