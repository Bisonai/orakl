import {
  Controller,
  Get,
  HttpCode,
  HttpStatus,
  Param,
  Put,
} from "@nestjs/common";
import { ApiOperation, ApiTags } from "@nestjs/swagger";
import { AccountsService } from "./accounts.service";

@Controller("accounts")
@ApiTags("accounts")
export class AccountsController {
  constructor(private readonly accountsService: AccountsService) {}

  @Get()
  @ApiOperation({ operationId: "getAccounts" })
  async findPage() {
    return this.accountsService.getAccountList();
  }

  @Get(":account")
  @ApiOperation({ operationId: "getAccount" })
  async findOne(@Param("account") account: string) {
    return this.accountsService.getAccountList();
  }

  @Put(":address/:name/:type")
  @ApiOperation({ operationId: "registerRedis" })
  @HttpCode(HttpStatus.OK)
  async insertAccount(
    @Param("address") address: string,
    @Param("name") name: string,
    @Param("type") type: string
  ) {
    return await this.accountsService.createAccount({ address, name, type });
  }
}
