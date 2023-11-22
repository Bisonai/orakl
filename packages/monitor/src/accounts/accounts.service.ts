import { Injectable } from "@nestjs/common";
import { AccountBalanceRepository } from "./accounts.repository";
import { Account } from "./entities/account.entity";

@Injectable()
export class AccountsService {
  constructor(
    private readonly accountBalanceRepository: AccountBalanceRepository,
  ) {}

  async getAccountList(): Promise<[Account] | null> {
    return await this.accountBalanceRepository.getAllAccount();
  }

  async createAccount(data: Account) {
    await this.accountBalanceRepository.insertAccount(data);
  }
}
