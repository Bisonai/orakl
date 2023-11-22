import { Injectable } from "@nestjs/common";
import { ServiceQueueCountInfo } from "./types";
import { BullsRepository } from "./bulls.repository";

import {
  Queue,
  QueueEvents,
  QueueEventsListener,
  RedisConnection,
  RedisOptions,
} from "bullmq";
import { QUEUE_ACTIVE_STATUS, QUEUE_STATUS, SERVICE } from "src/common/types";
import { RedisService } from "src/redis/redis.service";
import { OnApplicationBootstrap } from "@nestjs/common";
import { JobCompleted, JobFailed } from "./entities/job.data.entity";
import { IncomingWebhook } from "@slack/webhook";
import { MonitorConfigService } from "src/monitor.config/monitor.config.service";
import {
  AggregatorJobCompleted,
  AggregatorJobFailed,
} from "./entities/aggregator.job.data.entity";
import { QueueDto, QueueUpdateDto } from "./entities/queue.entity";
import { Redis } from "ioredis";

@Injectable()
export class BullsService implements OnApplicationBootstrap {
  private queueEvents: { [index: string]: QueueEvents } = {};
  private availableQueueStatus: QUEUE_STATUS[] = [
    QUEUE_STATUS.COMPLETED,
    QUEUE_STATUS.FAILED,
  ];
  constructor(
    private readonly bullsRepository: BullsRepository,
    private readonly redisService: RedisService,
    private readonly monitorConfigService: MonitorConfigService
  ) {}

  async onApplicationBootstrap() {
    await this.onTriggerJobCompleted(SERVICE.VRF);
    await this.onTriggerJobCompleted(SERVICE.REQUEST_RESPONSE);
    await this.onTriggerJobCompleted(SERVICE.AGGREGATOR);
    await this.onTriggerJobFailed(SERVICE.VRF);
    await this.onTriggerJobFailed(SERVICE.AGGREGATOR);
    await this.onTriggerJobFailed(SERVICE.REQUEST_RESPONSE);
  }

  async getQueueCounts() {
    let data = await this.getQueueCountsByService(SERVICE.VRF);
    data.push(await this.getQueueCountsByService(SERVICE.REQUEST_RESPONSE));
    data.push(await this.getQueueCountsByService(SERVICE.AGGREGATOR));
    return data;
  }

  async getQueueCountsByService(serviceName: SERVICE) {
    let countsInfo = [];
    const queues = await this.findQueueList(serviceName);
    console.log(queues);
    await Promise.all(
      queues.map(async (d) => {
        const data: any = await this.getQueueCountsByQueue(serviceName, d.name);
        const status = await this.getQueueStatus(serviceName, d.name);
        const queueData: ServiceQueueCountInfo = {
          service: serviceName,
          queue: d.name,
          status,
          ...data,
        };
        countsInfo.push(queueData);
      })
    );
    return countsInfo;
  }

  async getQueueStatus(
    serviceName: SERVICE,
    queueName: string
  ): Promise<boolean> {
    return await this.bullsRepository.findStatusListByServiceAndQueue(
      serviceName,
      queueName
    );
  }

  async getRedisInfo() {
    let data = [];
    data.push(await this.getRedisInfoByService(SERVICE.VRF));
    data.push(await this.getRedisInfoByService(SERVICE.REQUEST_RESPONSE));
    data.push(await this.getRedisInfoByService(SERVICE.AGGREGATOR));
    return data;
  }

  async getRedisInfoByService(serviceName: SERVICE) {
    const conn = await this.getRedisOptions(serviceName);
    const redis = new Redis(conn);
    const redisInfo = await redis.info();
    const usedMemoryHuman = parseInt(
      redisInfo.match(/used_memory_human:(\d+)/)[1],
      10
    );

    const versionInfo = redisInfo.match(/redis_version:(\d+\.\d+\.\d+)/);
    const redisVersion = versionInfo ? versionInfo[1] : "Unknown";
    const connectedClientsInfo = redisInfo.match(/connected_clients:(\d+)/);
    const connectedClients = connectedClientsInfo
      ? parseInt(connectedClientsInfo[1])
      : 0;

    const blockedClientsInfo = redisInfo.match(/blocked_clients:(\d+)/);
    const blockedClients = blockedClientsInfo
      ? parseInt(blockedClientsInfo[1])
      : 0;

    const fragmentationRatioInfo = redisInfo.match(
      /mem_fragmentation_ratio:(\d+\.\d+)/
    );
    const fragmentationRatio = fragmentationRatioInfo
      ? parseFloat(fragmentationRatioInfo[1])
      : 0;

    const commandsProcessedInfo = redisInfo.match(
      /total_commands_processed:(\d+)/
    );
    const commandsProcessed = commandsProcessedInfo
      ? parseInt(commandsProcessedInfo[1])
      : 0;

    const expiredKeysInfo = redisInfo.match(/expired_keys:(\d+)/);
    const expiredKeys = expiredKeysInfo ? parseInt(expiredKeysInfo[1]) : 0;

    const cpuUsageInfo = redisInfo.match(/used_cpu_sys:(\d+\.\d+)/);
    const cpuUsage = cpuUsageInfo ? parseFloat(cpuUsageInfo[1]) : 0;

    const uptimeInfo = redisInfo.match(/uptime_in_days:(\d+)/);
    const uptimeInDays = uptimeInfo ? parseInt(uptimeInfo[1]) : 0;

    return {
      serviceName,
      usedMemoryHuman,
      redisVersion,
      fragmentationRatio,
      connectedClients,
      blockedClients,
      commandsProcessed,
      expiredKeys,
      cpuUsage,
      uptimeInDays,
    };
  }

  async getQueueCountsByQueue(
    serviceName: SERVICE,
    queueName: string
  ): Promise<{
    [index: string]: number;
  }> {
    const conn = await this.getRedisOptions(serviceName);
    const queue = new Queue(queueName, { connection: conn });
    return await queue.getJobCounts();
  }

  async getListOfQueue(
    serviceName: SERVICE,
    queueName: string,
    queueStatus: QUEUE_STATUS
  ) {
    const conn = await this.getRedisOptions(serviceName);
    const queue = new Queue(queueName, { connection: conn });
    switch (queueStatus) {
      case QUEUE_STATUS.ACTIVE: {
        return await queue.getActive();
      }
      case QUEUE_STATUS.WAITING: {
        return await queue.getWaiting();
      }
      case QUEUE_STATUS.COMPLETED: {
        return await queue.getCompleted();
      }
      case QUEUE_STATUS.DELAYED: {
        return await queue.getDelayed();
      }
      case QUEUE_STATUS.FAILED: {
        return await queue.getFailed();
      }
    }
  }
  async activeQueueStatus(
    serviceName: SERVICE,
    queueName: string,
    status: QUEUE_ACTIVE_STATUS
  ) {
    const queue: QueueUpdateDto = {
      service: serviceName,
      name: queueName,
      status: status == "start" ? true : false,
    };
    const result = await this.bullsRepository.updateQueueStatus(queue);

    return this.onTriggerJobCompletedStatus(serviceName, queueName, status);
  }

  async findQueueList(service: SERVICE) {
    return this.bullsRepository.findAllQueueListByService(service);
  }

  async registerQueue(serviceName, queueName): Promise<string> {
    return await this.bullsRepository.create(serviceName, queueName);
  }

  async getRedisOptions(serviceName: SERVICE): Promise<RedisOptions> {
    return await this.redisService.findRedis(serviceName);
  }
  /**
   * @param service
   * @param queueName
   * @param status
   */
  async onTriggerJobCompletedStatus(
    service: SERVICE,
    queueName: string,
    status: QUEUE_ACTIVE_STATUS
  ) {
    if (status == QUEUE_ACTIVE_STATUS.STOP) {
      // console.log("stop trigger:", service);
      this.availableQueueStatus.map((status) => {
        const listeners =
          this.queueEvents?.[queueName]?.listeners(status) || [];
        listeners.map((listener) => {
          this.queueEvents[queueName].off(
            status,
            listener as QueueEventsListener[QUEUE_STATUS]
          );
        });
        this.queueEvents = { ...this.queueEvents, [queueName]: undefined };
      });
    } else {
      // console.log("start trigger:", service);
      if (!Boolean(this.queueEvents?.[queueName])) {
        this.queueEvents = {
          ...this.queueEvents,
          [queueName]: new QueueEvents(queueName, {
            connection: await this.getRedisOptions(service),
          }),
        };
      }
      const queue = new Queue(queueName, {
        connection: await this.getRedisOptions(service),
      });
      this.availableQueueStatus.map((status) => {
        this.queueEvents[queueName].on(
          status,
          this.getListener(status, service, queue)
        );
      });
    }
  }

  public getListener<ListenerType extends QUEUE_STATUS>(
    listenerType: ListenerType,
    service: SERVICE,
    queue: Queue
  ): QueueEventsListener[ListenerType] {
    const completedListener: QueueEventsListener[QUEUE_STATUS.COMPLETED] =
      async (
        args: {
          jobId: string;
          returnvalue: string;
          prev?: string;
        },
        id: string
      ) => {
        const data = await queue.getJob(args.jobId);
        const totalListener = Object.entries(this.queueEvents)
          .filter(([queueName, queueEvents]) => {
            return queueEvents?.listeners?.length > 0;
          })
          .reduce((acc, [queueName, queueEvents]) => {
            // console.log(
            //   `The queue ${queueName} has ${queueEvents?.listeners?.length} listeners.`
            // );
            return acc + queueEvents?.listeners?.length;
          }, 0);
        if (service == SERVICE.AGGREGATOR) {
          const dataSet: AggregatorJobCompleted = {
            service: service,
            name: data?.queueName,
            job_id: data?.id,
            job_name: data?.name,
            oracle_address: data?.data?.oracleAddress,
            delay: data?.data?.delay,
            round_id: data?.data?.roundId,
            worker_source: data?.data?.workerSource,
            submission: data?.data?.submission,
            data_set: data?.data,
            added_at: data?.timestamp,
            process_at: data?.processedOn,
            completed_at: data?.finishedOn,
          };
          await this.bullsRepository.createJobLog(
            dataSet,
            service,
            QUEUE_STATUS.COMPLETED
          );
        } else {
          const dataSet: JobCompleted = {
            service: service,
            name: data?.queueName,
            job_id: data?.id,
            job_name: data?.name,
            contract_address: data?.data?.contractAddress,
            block_number: data?.data?.blockNumber,
            block_hash: data?.data?.balckHash,
            callback_address: data?.data?.callbackAddress,
            block_num: data?.data?.blockNum,
            request_id: data?.data?.requestId,
            acc_id: data?.data?.accId,
            pk: data?.data?.pk,
            seed: data?.data?.seed,
            proof: data?.data?.proof,
            u_point: data?.data?.uPoint,
            pre_seed: data?.data?.preSeed,
            num_words: data?.data?.numWords,
            v_components: data?.data?.vComponents,
            callback_gas_limit: data?.data?.callbackGasLimit,
            sender: data?.data?.sender,
            is_direct_payment: data?.data?.isDirectPayment,
            event: data?.data?.event,
            data_set: data?.data,
            data: data?.data?.data,
            added_at: data?.timestamp,
            process_at: data?.processedOn,
            completed_at: data?.finishedOn,
          };
          await this.bullsRepository.createJobLog(
            dataSet,
            service,
            QUEUE_STATUS.COMPLETED
          );
        }
      };
    const failedListener: QueueEventsListener[QUEUE_STATUS.FAILED] = async (
      args: {
        jobId: string;
        failedReason: string;
        prev?: string;
      },
      id: string
    ) => {
      console.log("There is a failed event:", args);
      const data = await queue.getJob(args.jobId);
      console.log("data:", data);
      const totalListener = Object.entries(this.queueEvents)
        .filter(([queueName, queueEvents]) => {
          return queueEvents.listeners.length > 0;
        })
        .reduce((acc, [queueName, queueEvents]) => {
          // console.log(
          //   `The queue ${queueName} has ${queueEvents.listeners.length} listeners.`
          // );
          return acc + queueEvents.listeners.length;
        }, 0);
      console.log("serviceName:", service);
      console.log(`Total listeners: ${totalListener}`);
      if (service == SERVICE.AGGREGATOR) {
        const dataSet: AggregatorJobFailed = {
          error: data?.stacktrace,
          service: service,
          name: data?.queueName,
          job_id: data?.id,
          job_name: data?.name,
          oracle_address: data?.data?.oracleAddress,
          delay: data?.data?.delay,
          round_id: data?.data?.roundId,
          worker_source: data?.data?.workerSource,
          submission: data?.data?.submission,
          data_set: data?.data,
          added_at: data?.timestamp,
          process_at: data?.processedOn,
          completed_at: data?.finishedOn,
        };
        console.log("Aggregator Failed Job", dataSet);
        // send slack message
        this.sendToSlackFailedJob(dataSet);
        await this.bullsRepository.createJobLog(
          dataSet,
          service,
          QUEUE_STATUS.FAILED
        );
      } else {
        const dataSet: JobFailed = {
          error: data?.stacktrace,
          service: service,
          name: data?.queueName,
          job_id: data?.id,
          job_name: data?.name,
          contract_address: data?.data?.contractAddress,
          block_number: data?.data?.blockNumber,
          block_hash: data?.data?.balckHash,
          callback_address: data?.data?.callbackAddress,
          block_num: data?.data?.blockNum,
          request_id: data?.data?.requestId,
          acc_id: data?.data?.accId,
          pk: data?.data?.pk,
          seed: data?.data?.seed,
          proof: data?.data?.proof,
          u_point: data?.data?.uPoint,
          pre_seed: data?.data?.preSeed,
          num_words: data?.data?.numWords,
          v_components: data?.data?.vComponents,
          callback_gas_limit: data?.data?.callbackGasLimit,
          sender: data?.data?.sender,
          is_direct_payment: data?.data?.isDirectPayment,
          event: data?.data?.event,
          data_set: data?.data,
          data: data.data?.data,
          added_at: data?.timestamp,
          process_at: data?.processedOn,
          completed_at: data?.finishedOn,
        };
        // send slack message
        console.log("VRF or RequestResponse Failed Job", dataSet);
        try {
          await this.sendToSlackFailedJob(dataSet);
        } catch (e) {
          console.log(e);
        }
        await this.bullsRepository.createJobLog(
          dataSet,
          service,
          QUEUE_STATUS.FAILED
        );
      }
    };

    const listenerMap = {
      [QUEUE_STATUS.COMPLETED]: completedListener,
      [QUEUE_STATUS.FAILED]: failedListener,
    };

    return listenerMap[
      listenerType as keyof typeof listenerMap
    ] as QueueEventsListener[ListenerType];
  }

  async onTriggerJobCompleted(service: SERVICE) {
    // console.log("completed job triggered:", service);
    const connection = await this.getRedisOptions(service);
    const queues = await this.bullsRepository.findQueueListByService(
      service,
      true
    );
    queues.map((d) => {
      const queueEvents = new QueueEvents(d.name, {
        connection,
      });
      const queue = new Queue(d.name, { connection });
      queueEvents.on(
        QUEUE_STATUS.COMPLETED,
        this.getListener(QUEUE_STATUS.COMPLETED, service, queue)
      );
      this.queueEvents = { ...this.queueEvents, [d.name]: queueEvents };
    });
  }

  async onTriggerJobFailed(service: SERVICE) {
    const connection = await this.getRedisOptions(service);
    const queues = await this.bullsRepository.findQueueListByService(
      service,
      true
    );
    console.log("trggered Failed Job:", service);
    queues.map((d) => {
      const queueEvents = new QueueEvents(d.name, {
        connection,
      });
      const queue = new Queue(d.name, { connection });
      queueEvents.on(
        QUEUE_STATUS.FAILED,
        this.getListener(QUEUE_STATUS.FAILED, service, queue)
      );
    });
  }

  async sendToSlackFailedJob(data: JobFailed) {
    const { value } = await this.monitorConfigService.getValueByName(
      "slack_url"
    );
    const webhook = new IncomingWebhook(value);
    const now = new Date(); // 현재 시간
    const utcNow = now.getTime() + now.getTimezoneOffset() * 60 * 1000; // 현재 시간을 utc로 변환한 밀리세컨드값
    const koreaTimeDiff = 9 * 60 * 60 * 1000; // 한국 시간은 UTC보다 9시간 빠름(9시간의 밀리세컨드 표현)
    const koreaNow = new Date(utcNow + koreaTimeDiff); // utc로 변환된 값을 한국 시간으로 변환시키기 위해 9시간(밀리세컨드)를 더함
    const month = koreaNow.toLocaleString("en-US", { month: "long" });
    const day = koreaNow.getDate();
    const year = koreaNow.getFullYear();
    const hour = koreaNow.getHours();
    const minute = koreaNow.getMinutes();

    const headerText = `:scream_cat:  Failed job in ${data.service}`;
    const dateText = `${month} ${day}, ${year} ${hour}:${minute}   |   Redis Report`;
    const queueNameText = `:herb: *${data.name}*`;
    const jobNameText = `\`job name\`  ${data.job_name}`;
    const jobIdText = `\`job id\`  ${data.job_id}`;
    let dataSet = JSON.stringify(data.data_set);
    if (dataSet) {
      dataSet = dataSet.replace(/"/g, "");
      dataSet = dataSet.replace("{", "`");
      dataSet = dataSet.replace(/:/g, "` ");
      dataSet = dataSet.replace("}", "");
      dataSet = dataSet.replace(/,/g, "\n`");
    }
    const dataSetText = `>>> ${dataSet}`;

    let stackTrace = data?.error;
    let stackTraceText: string;
    if (stackTrace) {
      stackTraceText = JSON.stringify(stackTrace[0]);
      if (stackTraceText) {
        stackTraceText = stackTraceText.replace(/"/g, "");
        stackTraceText = stackTraceText.replace(/\\n/g, "\n");
        stackTraceText = `\`\`\`${stackTraceText}\`\`\``;
      } else {
        stackTraceText = "null";
      }
    } else {
      stackTraceText = "null";
    }

    await webhook.send({
      blocks: [
        {
          type: "header",
          text: {
            type: "plain_text",
            text: headerText,
          },
        },
        {
          type: "context",
          elements: [
            {
              text: dateText,
              type: "mrkdwn",
            },
          ],
        },
        {
          type: "divider",
        },
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: queueNameText,
          },
        },
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: jobNameText,
          },
        },
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: jobIdText,
          },
        },
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: "*Data*",
            verbatim: false,
          },
        },
        {
          type: "context",
          elements: [
            {
              text: dataSetText,
              type: "mrkdwn",
            },
          ],
        },
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: "*Stack Trace*",
          },
        },
        {
          type: "context",
          elements: [
            {
              text: stackTraceText,
              type: "mrkdwn",
            },
          ],
        },
        {
          type: "divider",
        },
        {
          type: "context",
          elements: [
            {
              type: "image",
              image_url: "https://www.orakl.network/favicon.ico",
              alt_text: "orakl network",
            },
            {
              type: "mrkdwn",
              text: " Developed by Bisonai Infra Team",
            },
          ],
        },
        {
          type: "divider",
        },
      ],
    });
  }
}
