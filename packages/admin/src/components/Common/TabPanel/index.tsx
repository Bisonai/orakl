import { ITabPanelProps } from "@/utils/types";
import React from "react";
import { TabPanelBase } from "./styled";
import { useTabContext } from "@/hook/useTabContext";

const TabPanel = ({ tabId, children }: ITabPanelProps): JSX.Element => {
  const { activeTab } = useTabContext();

  return <TabPanelBase>{activeTab === tabId && children}</TabPanelBase>;
};

export default TabPanel;
