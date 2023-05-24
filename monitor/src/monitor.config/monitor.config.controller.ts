import { Controller, HttpCode, HttpStatus, Param, Put } from "@nestjs/common";
import { ApiOperation, ApiTags } from "@nestjs/swagger";
import { SERVICE } from "src/common/types";
import { MonitorConfigService } from "./monitor.config.service";

@Controller("config")
@ApiTags("config")
export class MonitorConfigController {
  constructor(private readonly monitorConfigService: MonitorConfigService) {}

  @Put("/register/:name/:value")
  @ApiOperation({ operationId: "registerConfig" })
  @HttpCode(HttpStatus.OK)
  async createRedis(
    @Param("name") name: string,
    @Param("value") value: string,
  ) {
    return await this.monitorConfigService.registerConfig(name, value);
  }
}
