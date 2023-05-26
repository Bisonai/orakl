import {
    Injectable,
    Inject,
    OnModuleInit,
    OnModuleDestroy,
} from "@nestjs/common";
import { Pool, QueryResult } from "pg";

    @Injectable()
    export class MonitorConfigRepository implements OnModuleInit, OnModuleDestroy {
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
    
    // Insert a new row into the 'config' table
    async createConfig(name, value) {
        const query = {
        text: 'INSERT INTO config(name, value) VALUES($1, $2) RETURNING *',
        values: [name, value],
        };
        const { rows } = await this.monitorDatabasePool.query(query);
        return rows[0];
    }
    
    // Get a row from the 'config' table by its 'id' column
    async getConfigById(id) {
        const query = {
        text: 'SELECT * FROM config WHERE id = $1',
        values: [id],
        };
    
        const { rows } = await this.monitorDatabasePool.query(query);
        return rows[0];
    }
    
    // Get a row from the 'config' table by its 'name' column
    async getConfigByName(name) {
        const query = {
        text: 'SELECT * FROM config WHERE name = $1',
        values: [name],
        };
        
        const { rows } = await this.monitorDatabasePool.query(query);
        return rows[0];
    }
        
    // Update a row in the 'config' table by its 'id' column
    async updateConfigById(id, name, value) {
        const query = {
        text: 'UPDATE config SET name = $1, value = $2 WHERE id = $3 RETURNING *',
        values: [name, value, id],
        };
    
        const { rows } = await this.monitorDatabasePool.query(query);
        return rows[0];
    }
    
    // Delete a row from the 'config' table by its 'id' column
    async deleteConfigById(id) {
        const query = {
        text: 'DELETE FROM config WHERE id = $1',
        values: [id],
        };
    
        await this.monitorDatabasePool.query(query);
    }
        
}
    