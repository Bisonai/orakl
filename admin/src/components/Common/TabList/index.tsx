import { ITabListProps, ITabProps } from "@/utils/types";
import React, { useCallback, useEffect } from "react";
import { LabelWithIconBase, TabBase, TabListBase } from "./styled";
import { useTabContext } from "@/hook/useTabContext";

const TabList = ({ tabs, ...props }: ITabListProps): JSX.Element => {
  const { activeTab, setActiveTab } = useTabContext();

  const handleClickTab = useCallback(
    (tabId: string) => {
      setActiveTab(tabId);
      const url = new URL(window.location.href);
      const searchParams = new URLSearchParams(url.search);
      searchParams.set("activetab", tabId);
      url.search = searchParams.toString();
      const newUrl = url.toString();
      window.history.replaceState({}, "", newUrl);
    },
    [setActiveTab]
  );
  useEffect(() => {
    const url = new URL(window.location.href);
    const activetab = url.searchParams.get("activetab");

    const validTabIds = tabs.map((tab) => tab.tabId);
    if (activetab && validTabIds.includes(activetab)) {
      setActiveTab(activetab);
    }
  }, [setActiveTab, tabs]);

  return (
    <TabListBase {...props}>
      {tabs.map((tab) => (
        <Tab
          {...tab}
          key={tab.tabId}
          onClick={handleClickTab}
          className={activeTab === tab.tabId ? "selected" : undefined}
          selected={activeTab === tab.tabId}
        />
      ))}
    </TabListBase>
  );
};

const Tab = ({
  tabId,
  label,
  tabIcon,
  selectedTabIcon,
  onClick,
  selected,
  ...props
}: ITabProps): JSX.Element => {
  const handleClick = useCallback(() => {
    onClick?.(tabId);
  }, [onClick, tabId]);

  return (
    <TabBase id={tabId} {...props} onClick={handleClick}>
      <LabelWithIcon
        tabIcon={selected ? selectedTabIcon || tabIcon : tabIcon}
        label={label}
      />
    </TabBase>
  );
};

const LabelWithIcon = ({
  label,
  tabIcon,
}: Omit<ITabProps, "tabId">): JSX.Element => {
  return (
    <LabelWithIconBase className="noselect">
      {tabIcon}
      {label}
    </LabelWithIconBase>
  );
};

export default TabList;
