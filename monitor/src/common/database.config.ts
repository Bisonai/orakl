import { Injectable } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';

@Injectable()
export class DatabaseConfigService {
  constructor(private readonly configService: ConfigService) {}

  get monitorDatabase() {
    return {
      user: this.configService.get<string>('database.monitor.user'),
      host: this.configService.get<string>('database.monitor.host'),
      database: this.configService.get<string>('database.monitor.database'),
      password: this.configService.get<string>('database.monitor.password'),
      port: parseInt(this.configService.get<string>('database.monitor.port'), 10) || 5432,
    };
  }

  get oraklDatabase() {
    return {
      user: this.configService.get<string>('database.orakl.user'),
      host: this.configService.get<string>('database.orakl.host'),
      database: this.configService.get<string>('database.orakl.database'),
      password: this.configService.get<string>('database.orakl.password'),
      port: parseInt(this.configService.get<string>('database.orakl.port'), 10) || 5432,
    };
  }

  // get graphNodeDatabase() {
  //   return {
  //     user: this.configService.get<string>('MONITOR_POSTGRES_USER'),
  //     host: this.configService.get<string>('MONITOR_POSTGRES_HOST'),
  //     database: this.configService.get<string>('MONITOR_POSTGRES_DATABASE'),
  //     password: this.configService.get<string>('MONITOR_POSTGRES_PASSWORD'),
  //     port: parseInt(this.configService.get<string>('MONITOR_POSTGRES_PORT'), 10) || 5432,
  //   };
  // }

}
