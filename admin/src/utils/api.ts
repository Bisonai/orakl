import type { Method } from "axios";
import authenticatedAxios from "@/lib/authenticatedAxios";
/**********************************************
 * API
 **********************************************/

const isDevelopment = process.env.NODE_ENV === "development";
const NEXT_PUBLIC_API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL;
const NEXT_PUBLIC_API_QUEUES_URL = process.env.NEXT_PUBLIC_API_QUEUES_URL;

export const api = {
  queues: () => `${NEXT_PUBLIC_API_QUEUES_URL}/queues`,
  queuesInfo: () => `${NEXT_PUBLIC_API_QUEUES_URL}/queues/info`,
  service: (serviceName: string) =>
    `${NEXT_PUBLIC_API_QUEUES_URL}/queues/${serviceName}`,
  queueName: ({
    serviceName,
    queueName,
  }: {
    serviceName: string;
    queueName: string;
  }) => `${NEXT_PUBLIC_API_QUEUES_URL}/queues/${serviceName}/${queueName}`,
  queueStatus: ({
    serviceName,
    queueName,
    status,
  }: {
    serviceName: string;
    queueName: string;
    status: string;
  }) =>
    `${NEXT_PUBLIC_API_QUEUES_URL}/queues/${serviceName}/${queueName}/${status}`,
  getConfigChain: () => `${NEXT_PUBLIC_API_BASE_URL}/api/v1/chain/`,
  getConfigService: () => `${NEXT_PUBLIC_API_BASE_URL}/api/v1/service/`,
  configChain: (id: string) => `${NEXT_PUBLIC_API_BASE_URL}/api/v1/chain/${id}`,
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
  return await authenticatedAxios.request({ method, url: url, params, data });
};
