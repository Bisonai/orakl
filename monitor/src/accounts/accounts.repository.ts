import { Injectable, OnModuleInit, OnModuleDestroy, Inject } from '@nestjs/common';
import { Pool } from 'pg';
import { AccountDTO } from './dto/account.dto';
import { BalanceDTO } from './dto/balance.dto';

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


    async getBalance(address: string): Promise<number | null> {
        const query = 'SELECT * FROM balance WHERE address = $1';
        const values = [address];
        const result = await this.monitorClient.query(query, values);
        return result.rows[0] ? result.rows[0].balance : null;
    }

    async getBalanceAlarmAmount(address: string): Promise<number | null> {
        const query = 'SELECT * FROM account WHERE address = $1';
        const values = [address];
        const result = await this.monitorClient.query(query, values);
        return result.rows[0] ? result.rows[0].balance_alarm_amount : null;
    }

    async upsertBalance(balance: BalanceDTO): Promise<void> {
        const query = `
        INSERT INTO balance(address, balance)
        VALUES ($1, $2)
        ON CONFLICT (address) DO UPDATE
        SET balance = EXCLUDED.balance
        `;
        const values = [balance.address, balance.balance];
        await this.monitorClient.query(query, values);
    }
}
