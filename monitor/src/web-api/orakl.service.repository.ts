import { Injectable, OnModuleInit, OnModuleDestroy, Inject } from '@nestjs/common';
import { Pool } from 'pg';
import { ErrorResultDto } from './dto/error.dto';


@Injectable()
export class OraklServiceRepository implements OnModuleInit, OnModuleDestroy {
  private oraklServiceClient: any;

  constructor(
    @Inject("ORAKL_DATABASE") private readonly oraklDatabasePool: Pool
  ) {}

  async onModuleInit() {
    this.oraklServiceClient = await this.oraklDatabasePool.connect();
  }

  async onModuleDestroy() {
    await this.oraklServiceClient.release();
  }

  async getErrorResoutByRequestId(
    chain: string,
    requestIds: any
  ): Promise<ErrorResultDto | null> {
    const parsedRequestIds = requestIds.slice(1, -1).split(",");
    console.log(parsedRequestIds);
    const query = {
      text: "SELECT * FROM error where request_id = ANY ($1::text[])",
      values: [parsedRequestIds.map((id) => id.toString())],
    };
    const result = await this.oraklServiceClient.query(query);
    return result.rows || null;
  }
}
