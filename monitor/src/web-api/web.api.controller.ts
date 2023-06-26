import {
    Controller,
    Get,
    Param,
  } from "@nestjs/common";
  import { ApiOperation, ApiTags } from "@nestjs/swagger";
import { WebApiService } from "./web.api.service";
import { ErrorResultDto } from "./dto/error.dto";

  
  @Controller("web")
  @ApiTags("web")
  export class WebApiController {
    constructor(private readonly webApiService: WebApiService) {}

    @Get(":chain/:requestId")
    @ApiOperation({ operationId: "getErrorResult" })
    async findOne(@Param("chain") chain: string, @Param("requestId") requestId: string): Promise<ErrorResultDto | null> {
      return await this.webApiService.getErrorResult(chain, requestId);
    }

  }
  