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
  getChainConfig: () => `${NEXT_PUBLIC_API_BASE_URL}/api/v1/chain`,
  getServiceConfig: () => `${NEXT_PUBLIC_API_BASE_URL}/api/v1/service`,
  getListenerConfig: () => `${NEXT_PUBLIC_API_BASE_URL}/api/v1/listener`,
  getVrfKeysConfig: () => `${NEXT_PUBLIC_API_BASE_URL}/api/v1/vrf`,
  getAdapterConfig: () => `${NEXT_PUBLIC_API_BASE_URL}/api/v1/adapter`,
  getAggregatorConfig: () => `${NEXT_PUBLIC_API_BASE_URL}/api/v1/aggregator`,
  getReporterConfig: () => `${NEXT_PUBLIC_API_BASE_URL}/api/v1/reporter`,
  modifyChainConfig: (id: string) =>
    `${NEXT_PUBLIC_API_BASE_URL}/api/v1/chain/${id}`,
  modifyServiceConfig: (id: string) =>
    `${NEXT_PUBLIC_API_BASE_URL}/api/v1/service/${id}`,
  modifyListenerConfig: (id: string) =>
    `${NEXT_PUBLIC_API_BASE_URL}/api/v1/listener/${id}`,
  modifyVrfKeysConfig: (id: string) =>
    `${NEXT_PUBLIC_API_BASE_URL}/api/v1/vrf/${id}`,
  modifyAdapterConfig: (id: string) =>
    `${NEXT_PUBLIC_API_BASE_URL}/api/v1/adapter/${id}`,
  modifyAggregatorConfig: (id: string) =>
    `${NEXT_PUBLIC_API_BASE_URL}/api/v1/aggregator/${id}`,
  modifyReporterConfig: (id: string) =>
    `${NEXT_PUBLIC_API_BASE_URL}/api/v1/reporter/${id}`,
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
  getChainConfig: { GET: any; POST: any };
  getServiceConfig: { GET: any; POST: any };
  getListenerConfig: { GET: any; POST: any };
  getVrfKeysConfig: { GET: any; POST: any };
  getAdapterConfig: { GET: any; POST: any };
  getAggregatorConfig: { GET: any; POST: any };
  getReporterConfig: { GET: any; POST: any };
  modifyChainConfig: { PATCH: any; DELETE: any };
  modifyServiceConfig: { PATCH: any; DELETE: any };
  modifyListenerConfig: { PATCH: any; DELETE: any };
  modifyVrfKeysConfig: { PATCH: any; DELETE: any };
  modifyAdapterConfig: { PATCH: any; DELETE: any };
  modifyAggregatorConfig: { PATCH: any; DELETE: any };
  modifyReporterConfig: { PATCH: any; DELETE: any };
}
export interface IApiData {
  queues: { GET: any };
  queuesInfo: { GET: any };
  service: { GET: any };
  queueName: { GET: any };
  queueStatus: { GET: any };
  getChainConfig: { GET: any; POST: any };
  getServiceConfig: { GET: any; POST: any };
  getListenerConfig: { GET: any; POST: any };
  getVrfKeysConfig: { GET: any; POST: any };
  getAdapterConfig: { GET: any; POST: any };
  getAggregatorConfig: { GET: any; POST: any };
  getReporterConfig: { GET: any; POST: any };
  modifyChainConfig: { PATCH: any; DELETE: any };
  modifyServiceConfig: { PATCH: any; DELETE: any };
  modifyListenerConfig: { PATCH: any; DELETE: any };
  modifyVrfKeysConfig: { PATCH: any; DELETE: any };
  modifyAdapterConfig: { PATCH: any; DELETE: any };
  modifyAggregatorConfig: { PATCH: any; DELETE: any };
  modifyReporterConfig: { PATCH: any; DELETE: any };
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
  route?: Parameters<IApi[T]>
) => {
  // @ts-ignore
  const url = route ? api[target](...route) : api[target]();
  const axios = await import("axios").then((m) => m.default);
  return await authenticatedAxios.request({ method, url: url, params, data });
};
