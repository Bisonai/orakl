import type { Method } from "axios";

/**********************************************
 * API
 **********************************************/

const isDevelopment = process.env.NODE_ENV === "development";

export const api = {
  queues: () => `http://localhost:8888/queues`,
  queuesInfo: () => `http://localhost:8888/queues/info`,
  service: (serviceName: string) =>
    `http://localhost:8888/queues/${serviceName}`,
  queueName: ({
    serviceName,
    queueName,
  }: {
    serviceName: string;
    queueName: string;
  }) => `http://localhost:8888/queues/${serviceName}/${queueName}`,
  queueStatus: ({
    serviceName,
    queueName,
    status,
  }: {
    serviceName: string;
    queueName: string;
    status: string;
  }) => `http://localhost:8888/queues/${serviceName}/${queueName}/${status}`,
  getConfigChain: () => `http://localhost:3030/api/v1/chain/`,
  getConfigService: () => `http://localhost:3030/api/v1/service/`,
  configChain: (id: string) => `http://localhost:3030/api/v1/chain/${id}`,
};

export type IApi = typeof api;

export type IApiBase = {
  [target in keyof IApi]: { [method in Method]: any | undefined };
};
export interface IApiParam {
  queues: { GET: any };
  queuesInfo: { GET: any };
  service: { GET: any };
  queueName: { GET: any };
  queueStatus: { GET: any };
  getConfigChain: { GET: any };
  getConfigService: { GET: any };
  configChain: { PATCH: any; DELETE: any };
}
export interface IApiData {
  queues: { GET: any };
  queuesInfo: { GET: any };
  service: { GET: any };
  queueName: { GET: any };
  queueStatus: { GET: any };
  getConfigChain: { GET: any };
  getConfigService: { GET: any };
  configChain: { PATCH: any; DELETE: any };
}

export const fetchInternalApi = async <
  T extends keyof IApi,
  M extends keyof IApiParam[T] & Method
>(
  {
    target,
    method,
    params,
    data,
  }: {
    target: T;
    method: M;
    params?: IApiParam[T][M];
    data?: IApiData[T][M];
  },
  route: Parameters<IApi[T]>
) => {
  // @ts-ignore
  const url = route ? api[target](...route) : api[target]();
  const axios = await import("axios").then((m) => m.default);
  return await axios.request({ method, url: url, params, data });
};
