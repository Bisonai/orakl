import {
  Controller,
  Get,
  HttpCode,
  HttpStatus,
  Param,
  Put,
  UseGuards,
} from "@nestjs/common";
import { ApiOperation, ApiTags } from "@nestjs/swagger";
import { AccountsService } from "./accounts.service";
import { AuthGuard } from "src/auth/auth.guard";

@Controller("accounts")
@ApiTags("accounts")
export class AccountsController {
  constructor(private readonly accountsService: AccountsService) {}

  @Get()
  @UseGuards(AuthGuard)
  @ApiOperation({ operationId: "getAccounts" })
  async findPage() {
    return this.accountsService.getAccountList();
  }

  @Get(":account")
  @UseGuards(AuthGuard)    
  @ApiOperation({ operationId: "getAccount" })
  async findOne(@Param("account") account: string) {
    return this.accountsService.getAccountList();
  }

  @Put(":address/:name/:type")
  @UseGuards(AuthGuard)    
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
