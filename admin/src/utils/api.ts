import type { Method } from "axios";

/**********************************************
 * API
 **********************************************/

const isDevelopment = process.env.NODE_ENV === "development";

export const api = {
  queues: () => "https://monitor-api.orakl.bisonai.net/queues",
  service: (serviceName: string) =>
    `https://monitor-api.orakl.bisonai.net/queues/${serviceName}`,
  queueName: ({
    serviceName,
    queueName,
  }: {
    serviceName: string;
    queueName: string;
  }) =>
    `https://monitor-api.orakl.bisonai.net/queues/${serviceName}/${queueName}`,
  queueStatus: ({
    serviceName,
    queueName,
    status,
  }: {
    serviceName: string;
    queueName: string;
    status: string;
  }) =>
    `https://monitor-api.orakl.bisonai.net/queues/${serviceName}/${queueName}/${status}`,
};

export type IApi = typeof api;

export type IApiBase = {
  [target in keyof IApi]: { [method in Method]: any | undefined };
};
export interface IApiParam {
  queues: { GET: any };
  service: { GET: any };
  queueName: { GET: any };
  queueStatus: { GET: any };
}
export interface IApiData {
  queues: { GET: any };
  service: { GET: any };
  queueName: { GET: any };
  queueStatus: { GET: any };
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
