import { Injectable } from "@nestjs/common";
import { PoolClient } from "pg";
import { CommonConfigService } from "src/common/common.config";
import { OraklServiceRepository } from "./orakl.service.repository";
import { ErrorResultDto } from "./dto/error.dto";

@Injectable()
export class WebApiService {
  private webApiClient: PoolClient;

  constructor(
    private readonly webApiRepository: OraklServiceRepository,
    private readonly commonConfigService: CommonConfigService
  ) {}

  async getErrorResult(
    chain: string,
    requestIds
  ): Promise<ErrorResultDto | null> {
    return await this.webApiRepository.getErrorResoutByRequestId(
      chain,
      requestIds
    );
  }
}
