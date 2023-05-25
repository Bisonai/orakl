import { Injectable, Inject, OnModuleInit, OnModuleDestroy } from '@nestjs/common';
import { Pool, QueryResult } from 'pg';
import { QueueDto, QueueUpdateDto } from './entities/queue.entity';
import { JobCompleted, JobFailed } from "./entities/job.data.entity";
import { QUEUE_STATUS, SERVICE } from "src/common/types";
import { AggregatorJobCompleted, AggregatorJobFailed } from './entities/aggregator.job.data.entity';

@Injectable()
export class BullsRepository implements OnModuleInit, OnModuleDestroy {
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

  async create(serviceName: string, queueName: string): Promise<string> {
    const result = await this.monitorClient.query(
      'SELECT * FROM "queue" WHERE "service" = $1 and "name" = $2',
      [serviceName, queueName]
    );

    if (result.rowCount === 0) {
      await this.monitorClient.query(
        'INSERT INTO "queue" (service, name) VALUES ($1, $2)',
        [serviceName, queueName]
      );
      return `Queue ${queueName} created`;
    } else {
      return `Queue ${queueName} is already exist`;
    }
  }

  async findAll(): Promise<QueueDto[]> {
    const query = `
        SELECT *
        FROM queue
        `;
    const result: QueryResult<QueueDto> = await this.monitorClient.query(query);
    return result.rows;
  }

  async findAllQueueListByService(service: SERVICE): Promise<QueueDto[]> {
    const query = `
        SELECT *
        FROM queue Where service = $1
        `;
    const result: QueryResult<QueueDto> = await this.monitorClient.query(
      query,
      [service]
    );
    return result.rows;
  }

  async findQueueListByService(
    service: SERVICE,
    status: boolean
  ): Promise<QueueDto[]> {
    const query = `
        SELECT *
        FROM queue Where service = $1 and status = $2
        `;
    const result: QueryResult<QueueDto> = await this.monitorClient.query(
      query,
      [service, status]
    );
    return result.rows;
  }

  async findStatusListByServiceAndQueue(
    service: SERVICE,
    queueName: string
  ): Promise<boolean> {
    const query = `
        SELECT status
        FROM queue Where service = $1 and name = $2
        `;
    const result = await this.monitorClient.query(query, [service, queueName]);
    return result.rows[0].status;
  }

  async findOne(idx: number): Promise<QueueDto> {
    const query = `
        SELECT *
        FROM queue
        WHERE idx = $1
        `;
    const values = [idx];
    const result: QueryResult<QueueDto> = await this.monitorClient.query(
      query,
      values
    );
    return result.rows[0];
  }

  async update(idx: number, updateQueueDto: QueueDto): Promise<QueueDto> {
    const { name, service, status } = updateQueueDto;
    const query = `
        UPDATE queue
        SET name = $1, service = $2
        WHERE idx = $3
        RETURNING *
        `;
    const values = [name, service, status, idx];
    const result: QueryResult<QueueDto> = await this.monitorClient.query(
      query,
      values
    );
    return result.rows[0];
  }

  async updateQueueStatus(queue: QueueUpdateDto): Promise<QueueDto> {
    const { name, service, status } = queue;
    const query = `
        UPDATE queue
        SET status = $1
        WHERE service = $2 and name = $3
        RETURNING *
        `;
    const values = [status, service, name];
    const result: QueryResult<QueueDto> = await this.monitorClient.query(
      query,
      values
    );
    return result.rows[0];
  }

  async remove(idx: number): Promise<void> {
    const query = `
        DELETE FROM queue
        WHERE idx = $1
        `;
    const values = [idx];
    await this.monitorClient.query(query, values);
  }

  async findCompleted(name: string, jobId: string, service: SERVICE) {
    const values = [name, jobId];

    if (service == SERVICE.VRF) {
      const query = `
      SELECT *
      FROM redis_vrf_completed 
      WHERE name = $1 and job_id = $2
      `;
      return await this.monitorClient.query(query, values);
    } else if (service == SERVICE.REQUEST_RESPONSE) {
      const query = `
      SELECT *
      FROM redis_request_response_completed 
      WHERE name = $1 and job_id = $2
      `;
      return await this.monitorClient.query(query, values);
    } else {
      return [];
    }
  }

  async findFailed(name: string, jobId: string, service: SERVICE) {
    const values = [name, jobId];

    if (service == SERVICE.VRF) {
      const query = `
      SELECT *
      FROM redis_vrf_failed
      WHERE name = $1 and job_id = $2
      `;
      return await this.monitorClient.query(query, values);
    } else if (service == SERVICE.REQUEST_RESPONSE) {
      const query = `
      SELECT *
      FROM redis_request_response_failed
      WHERE name = $1 and job_id = $2
      `;
      return await this.monitorClient.query(query, values);
    } else {
      return [];
    }
  }

  async createJobLog(
    jobData:
      | JobCompleted
      | JobFailed
      | AggregatorJobCompleted
      | AggregatorJobFailed,
    service: SERVICE,
    status: QUEUE_STATUS
  ): Promise<void> {
    const { name, job_id } = jobData;
    let tableMid: string;
    let tablePostfix: string;
    if (service == SERVICE.VRF) {
      tableMid = "vrf";
    } else if (service == SERVICE.REQUEST_RESPONSE) {
      tableMid = "request_response";
    } else if (service == SERVICE.AGGREGATOR) {
      tableMid = "aggregator";
    }
    if (status == QUEUE_STATUS.COMPLETED) {
      tablePostfix = "completed";
    } else if (status == QUEUE_STATUS.FAILED) {
      tablePostfix = "failed";
    }
    // console.log("service:", service);
    // console.log("status:", status);
    const tableName = "redis_" + tableMid + "_" + tablePostfix;
    // console.log("table name:", tableName);
    // console.log("queue name:", name);

    if (status == QUEUE_STATUS.COMPLETED) {
      if (service == SERVICE.AGGREGATOR) {
        await this.monitorClient.query(
          `INSERT INTO ${tableName} (service, name, job_id, job_name, oracle_address, delay, round_id, worker_source, submission, data_set, added_at, process_at, completed_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
          [...Object.values(jobData)]
        );
      } else {
        await this.monitorClient.query(
          `INSERT INTO ${tableName} (service, name, job_id, job_name, contract_address, block_number, block_hash, callback_address, block_num, request_id, acc_id, pk, seed, proof, u_point, pre_seed, num_words, v_components, callback_gas_limit, sender, is_direct_payment, event, data_set, data, added_at, process_at, completed_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27)`,
          [...Object.values(jobData)]
        );
      }
    }
    if (status == QUEUE_STATUS.FAILED) {
      if (service == SERVICE.AGGREGATOR) {
        await this.monitorClient.query(
          `INSERT INTO ${tableName} (error, service, name, job_id, job_name, oracle_address, delay, round_id, worker_source, submission, data_set, added_at, process_at, completed_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
          [...Object.values(jobData)]
        );
      } else {
        await this.monitorClient.query(
          `INSERT INTO ${tableName} (error, service, name, job_id, job_name, contract_address, block_number, block_hash, callback_address, block_num, request_id, acc_id, pk, seed, proof, u_point, pre_seed, num_words, v_components, callback_gas_limit, sender, is_direct_payment, event, data_set, data, added_at, process_at, completed_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28)`,
          [...Object.values(jobData)]
        );
      }
    }
  }
} 