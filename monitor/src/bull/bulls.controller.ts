import {
  Body,
  Controller,
  Get,
  HttpCode,
  HttpStatus,
  Param,
  Post,
  Put,
} from "@nestjs/common";
import { ApiOperation, ApiTags } from "@nestjs/swagger";
import { BullsService } from "./bulls.service";
import { QUEUE_ACTIVE_STATUS, QUEUE_STATUS, SERVICE } from "src/common/types";

@Controller("queues")
@ApiTags("queues")
export class BullsController {
  constructor(private readonly bullsService: BullsService) {}

  @Get("/")
  @ApiOperation({ operationId: "getQueueCounts" })
  async getCounts() {
    return await this.bullsService.getQueueCounts();
  }

  @Get("/info")
  @ApiOperation({ operationId: "getInfo" })
  async getRedisInfo() {
    return await this.bullsService.getRedisInfo();
  }

  @Get("/:service_name")
  @ApiOperation({ operationId: "getQueueCounts" })
  async getCountsByService(@Param("service_name") serviceName: SERVICE) {
    return await this.bullsService.getQueueCountsByService(serviceName);
  }

  @Get("/:service_name/:queue_name")
  @ApiOperation({ operationId: "getQueueCounts" })
  async getCountsByQueue(
    @Param("service_name") serviceName: SERVICE,
    @Param("queue_name")
    queueName: string
  ): Promise<{
    [index: string]: number;
  }> {
    return await this.bullsService.getQueueCountsByQueue(
      serviceName,
      queueName
    );
  }

  @Get("/:service_name/:queue_name/:status")
  @ApiOperation({ operationId: "getQueueList" })
  @HttpCode(HttpStatus.OK)
  async getQueueList(
    @Param("service_name") serviceName: SERVICE,
    @Param("queue_name")
    queueName: string,
    @Param("status") queueStatus: QUEUE_STATUS
  ) {
    return await this.bullsService.getListOfQueue(
      serviceName,
      queueName,
      queueStatus
    );
  }
  @Put("/:service_name/:queue_name/:active_status")
  @ApiOperation({ operationId: "getQueueList" })
  @HttpCode(HttpStatus.OK)
  async activeQueueStatus(
    @Param("service_name") serviceName: SERVICE,
    @Param("queue_name")
    queueName: string,
    @Param("active_status") active_status: QUEUE_ACTIVE_STATUS
  ) {
    return await this.bullsService.activeQueueStatus(
      serviceName,
      queueName,
      active_status
    );
  }
  @Post("/register/:service_name/:queue_name")
  @ApiOperation({ operationId: "registerQueue" })
  @HttpCode(HttpStatus.OK)
  async createOrUpdateQueue(
    @Param("service_name") serviceName: string,
    @Param("queue_name") queueName: string
  ) {
    console.log("service:", serviceName);
    console.log("name:", queueName);
    return await this.bullsService.registerQueue(serviceName, queueName);
  }
}
