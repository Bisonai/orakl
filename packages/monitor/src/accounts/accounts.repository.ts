import { Injectable, OnModuleInit, OnModuleDestroy, Inject } from '@nestjs/common';
import { Pool } from 'pg';
import { AccountDTO } from './dto/account.dto';

@Injectable()
export class AccountBalanceRepository implements OnModuleInit, OnModuleDestroy {
    private monitorClient: any;

    constructor(
        @Inject('MONITOR_DATABASE') private readonly monitorDatabasePool: Pool,
    ) {}

    async onModuleInit() {
        this.monitorClient = await this.monitorDatabasePool.connect();
    }

    async onModuleDestroy() {
        await this.monitorClient.release();
    }

    async getAllAccount(): Promise<[AccountDTO] | null> {
        const query = 'SELECT * FROM account';
        const result = await this.monitorClient.query(query);
        return result.rows || null;
    }

    async getAccount(address: string): Promise<AccountDTO | null> {
        const query = 'SELECT * FROM account WHERE address = $1';
        const values = [address];
        const result = await this.monitorClient.query(query, values);
        return result.rows[0] || null;
    }

    async insertAccount(account: AccountDTO): Promise<void> {
        const query = 'INSERT INTO account(address, name, type) VALUES ($1, $2, $3)';
        const values = [account.address, account.name, account.type];
        await this.monitorClient.query(query, values);
    }

    async updateAccount(account: AccountDTO): Promise<void> {
        const query = 'UPDATE account SET name = $1, type = $2 WHERE address = $3';
        const values = [account.name, account.type, account.address];
        await this.monitorClient.query(query, values);
    }
}
