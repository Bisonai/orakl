export interface IObject<T> {
  [key: string]: T;
}

export interface ITableHeaderProps {
  version: string;
  memoryUsage: string;
  fragmentationRatio: string;
  connectedClients: string;
  blockedClients: string;
  buttonText: string;
  onRefresh: () => void;
}

export interface IQueueData {
  service: string;
  queue: string;
  active: number;
  completed: number;
  delayed: number;
  failed: number;
  paused: number;
  waiting: number;
  "waiting-children": number;
}
