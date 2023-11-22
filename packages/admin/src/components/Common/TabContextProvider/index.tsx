import { ITabContextProps, ITabContextProviderProps } from "@/utils/types";
import React, { useState } from "react";

const TabContext: React.Context<ITabContextProps> =
  React.createContext<ITabContextProps>({
    activeTab: "",
    setActiveTab: function () {
      return;
    },
  });

const TabContextProvider = ({
  initTab,
  children,
}: ITabContextProviderProps): JSX.Element => {
  const [activeTab, setActiveTab] = useState<string>(initTab);

  return (
    <TabContext.Provider
      value={{
        activeTab,
        setActiveTab,
      }}
    >
      {children}
    </TabContext.Provider>
  );
};

export default TabContextProvider;

export { TabContext };
