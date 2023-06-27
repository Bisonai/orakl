import { Injectable } from "@nestjs/common";
import { PoolClient } from "pg";
import { ethers } from "ethers";
import { CommonConfigService } from "src/common/common.config";

import { MonitorConfigService } from "src/monitor.config/monitor.config.service";
import { MONITOR_CONFIG } from "src/common/types";
import { IncomingWebhook } from "@slack/webhook";
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
