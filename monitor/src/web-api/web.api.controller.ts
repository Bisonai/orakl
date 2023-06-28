import { Controller, Get, Param, Query } from "@nestjs/common";
import { ApiOperation, ApiTags } from "@nestjs/swagger";
import { WebApiService } from "./web.api.service";
import { ErrorResultDto } from "./dto/error.dto";
import { RquestDto } from "./dto/request.dto";

@Controller("web")
@ApiTags("web")
export class WebApiController {
  constructor(private readonly webApiService: WebApiService) {}

  @Get(":chain")
  @ApiOperation({ operationId: "getErrorResult" })
  async findOne(
    @Param("chain") chain: string,
    @Query() query: RquestDto
  ): Promise<ErrorResultDto | null> {
    const arrayParam = query.query;
    return await this.webApiService.getErrorResult(chain, arrayParam);
  }
}
