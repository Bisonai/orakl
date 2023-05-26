import { Injectable } from '@nestjs/common'
import { Cron } from "@nestjs/schedule";
import { AccountsService } from "./accounts/accounts.service";

@Injectable()
export class AppService {
  constructor(private readonly accountsService: AccountsService) {}
  health(): string {
    return "ok";
  }

  @Cron("*/10 * * * * *") // 매 10초마다 Balance Update
  accountBalanceCron() {
    this.accountsService.cronUpdateBalance();
  }

  @Cron("0 5 * * * *") // 매 1시간마다 5분에 실행 
  accountAlramLowBalance() {
    this.accountsService.cronAlarmLowBalance();
  }  
}
