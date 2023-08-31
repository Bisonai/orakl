import { Injectable } from '@nestjs/common'
import { AccountsService } from "./accounts/accounts.service";

@Injectable()
export class AppService {
  constructor(private readonly accountsService: AccountsService) {}
  health(): string {
    return "ok";
  }
}
