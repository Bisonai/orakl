import { StyledComponentProps } from "@mui/material";
import { ReactNode } from "react";

export interface IObject<T> {
  [key: string]: T;
}

export interface ITableHeaderProps {
  onRefresh: () => void;
  serviceData: IQueueInfoData;
}

export interface IQueueInfoData {
  blockedClients: number;
  commandsProcessed?: number;
  connectedClients: number;
  cpuUsage: number;
  expiredKeys?: number;
  fragmentationRatio: number;
  redisVersion: string;
  serviceName: string;
  uptimeInDays?: number;
  usedMemoryHuman?: number;
}

export interface IQueueData {
  service: string;
  status: boolean;
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

/** Tab */

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
  { tabId: "active", label: "Active" },
  { tabId: "waiting", label: "Waiting" },
  { tabId: "completed", label: "Completed" },
  { tabId: "failed", label: "Failed" },
  { tabId: "delayed", label: "Delayed" },
  { tabId: "paused", label: "Paused" },
];

/** Toast */

export interface IToast {
  id?: string | number;
  title: React.ReactNode;
  content: React.ReactNode;
  type: ToastType;
}

export enum ToastType {
  SUCCESS = "SUCCESS",
  ERROR = "ERROR",
}

export interface IToastContextProps {
  toasts: IToast[];
  addToast: (toast: IToast) => void;
  removeToast: (id: string | number) => void;
  updateToast: (toast: IToast) => void;
  clearToast: () => void;
}

/** Configuration */

export interface IConfigurationProps {
  chain: string;
  service: string;
  listener: string;
  vrfKeys: string;
  adapter: string;
  aggregator: string;
  reporter: string;
  fetcher: string;
  delegator: string;
}
export interface IDimmedPopupProps {
  title: string;
  confirmText?: string;
  cancelText?: string;
  onConfirm: (inputValue?: string) => void;
  onCancel: () => void;
  buttonTwo: boolean;
  form: boolean;
  size?: "small" | "medium" | "large";
}

export interface IDimmedPopupContext {
  isOpen: boolean;
  openDimmedPopup: (props: IDimmedPopupProps) => void;
  closeDimmedPopup: () => void;
  inputValue: string;
  setInputValue: React.Dispatch<React.SetStateAction<string>>;
}
