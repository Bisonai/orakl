export interface QueueCountInfo {
  waiting: number;
  active: number;
  completed: number;
  failed: number;
  delayed: number;
}
  
export interface ServiceQueueCountInfo extends QueueCountInfo {
  service: string;
  queue: string;
}