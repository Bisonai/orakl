import {
  Injectable,
  Inject,
  OnModuleInit,
  OnModuleDestroy,
} from "@nestjs/common";
import { Pool, QueryResult } from "pg";
import { QueueDto } from "../bull/entities/queue.entity";
import { RedisDto } from "./entities/redis.entity";
import { SERVICE } from "src/common/types";

@Injectable()
export class RedisRepository implements OnModuleInit, OnModuleDestroy {
  private monitorClient: any;

  constructor(
    @Inject("MONITOR_DATABASE") private readonly monitorDatabasePool: Pool
  ) {}

  async onModuleInit() {
    this.monitorClient = await this.monitorDatabasePool.connect();
  }

  async onModuleDestroy() {
    await this.monitorClient.release();
  }

  async create(
    serviceName: SERVICE,
    host: string,
    port: number
  ): Promise<string> {
    const result = await this.monitorClient.query(
      'SELECT * FROM "redis" WHERE "service" = $1 and "host" = $2 and "port" = $3 ',
      [serviceName, host, port]
    );

    if (result.rowCount === 0) {
      await this.monitorClient.query(
        'INSERT INTO "redis" (service, host, port) VALUES ($1, $2, $3)',
        [serviceName, host, port]
      );
      return `Redis ${serviceName} ${host}:${port} created`;
    } else {
      return `Redis ${serviceName} ${host}:${port} is already exist`;
    }
  }

  async findAll(): Promise<QueueDto[]> {
    const query = `
        SELECT *
        FROM redis
        `;
    const result: QueryResult<QueueDto> = await this.monitorClient.query(query);
    return result.rows;
  }

  async findOne(serviceName: string): Promise<RedisDto> {
    const query = `
        SELECT *
        FROM redis
        WHERE service = $1
        `;
    const values = [serviceName];
    const result: QueryResult<RedisDto> = await this.monitorClient.query(
      query,
      values
    );
    return result.rows[0];
  }

  async update(
    serviceName: string,
    updateRedisDto: RedisDto
  ): Promise<RedisDto> {
    const { host, port } = updateRedisDto;
    const query = `
        UPDATE redis
        SET host = $1, port = $2
        WHERE service = $3
        RETURNING *
        `;
    const values = [host, port, serviceName];
    const result: QueryResult<RedisDto> = await this.monitorClient.query(
      query,
      values
    );
    return result.rows[0];
  }

  async remove(serviceName: string): Promise<void> {
    const query = `
        DELETE FROM redis
        WHERE service = $1
        `;
    const values = [serviceName];
    await this.monitorClient.query(query, values);
  }
}
