import { StyledComponentProps } from "@mui/material";
import { ReactNode } from "react";

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

export interface IAccordionState {
  configuration: boolean;
  bull: boolean;
  account: boolean;
}

export interface ITabContextProps {
  activeTab: string;
  setActiveTab: (activeTab: string) => void;
}
export interface ITabContextProviderProps {
  initTab: string;
  children: ReactNode;
}

export interface ITabListProps {
  tabs: ITabProps[];
}

export interface ITabProps extends StyledComponentProps<"li"> {
  tabId: string;
  label: ReactNode;
  tabIcon?: ReactNode;
  queue?: IQueueData;
  selectedTabIcon?: ReactNode;
  onClick?: (tabId: string) => void;
  selected?: boolean;
  className?: string;
}

export interface ITabPanelProps {
  tabId: string;
  children: ReactNode;
}

export type StatusTab = {
  tabId: string;
  label: string;
};

export const statusTabs: StatusTab[] = [
  { tabId: "Active", label: "Active" },
  { tabId: "Waiting", label: "Waiting" },
  { tabId: "Completed", label: "Completed" },
  { tabId: "Failed", label: "Failed" },
  { tabId: "Delayed", label: "Delayed" },
  { tabId: "Paused", label: "Paused" },
];
