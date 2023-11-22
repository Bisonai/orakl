import { Module } from "@nestjs/common";
import { ConfigModule } from "@nestjs/config";
import { Pool } from "pg";
import { DatabaseConfigService } from "src/common/database.config";

@Module({
  imports: [ConfigModule],
  providers: [
    DatabaseConfigService,
    {
      provide: "MONITOR_DATABASE",
      useFactory: async (configService: DatabaseConfigService) => {
        const pool = new Pool(configService.monitorDatabase);
        await pool.connect();
        return pool;
      },
      inject: [DatabaseConfigService],
    },
    {
      provide: "ORAKL_DATABASE",
      useFactory: async (configService: DatabaseConfigService) => {
        const pool = new Pool(configService.oraklDatabase);
        await pool.connect();
        return pool;
      },
      inject: [DatabaseConfigService],
    },    
  ],
  exports: ["MONITOR_DATABASE", "ORAKL_DATABASE"],
})
export class DatabaseModule {}
