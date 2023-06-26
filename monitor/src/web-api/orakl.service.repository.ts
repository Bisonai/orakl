import { Injectable, OnModuleInit, OnModuleDestroy, Inject } from '@nestjs/common';
import { Pool } from 'pg';
import { ErrorResultDto } from './dto/error.dto';


@Injectable()
export class OraklServiceRepository implements OnModuleInit, OnModuleDestroy {
    private oraklServiceClient: any;

    constructor(
        @Inject('ORAKL_DATABASE') private readonly oraklDatabasePool: Pool,
    ) {}

    async onModuleInit() {
        this.oraklServiceClient = await this.oraklDatabasePool.connect();
    }

    async onModuleDestroy() {
        await this.oraklServiceClient.release();
    }

    async getErrorResoutByRequestId(chain: string, requestId: string): Promise<ErrorResultDto | null> {
        const query = {
            text: 'SELECT * FROM error where request_id = $1',
            values: [requestId]
        }
        const result = await this.oraklServiceClient.query(query);
        return result.rows || null;
    }
}
