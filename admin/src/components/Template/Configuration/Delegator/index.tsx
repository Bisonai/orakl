import { fetchInternalApi } from "@/utils/api";
import React from "react";
import { useQuery } from "react-query";
import TabContextProvider from "@/components/Common/TabContextProvider";
import TabList from "@/components/Common/TabList";
import TabPanel from "@/components/Common/TabPanel";
import { StatusTab, delegatorTabs } from "@/utils/types";
import { DataListBase, DelegatorContainer, TitleBase } from "./styled";
import { IsLoadingBase } from "@/components/Template/BullMonitor/DetailTable/styled";

const Delegator = () => {
  const organizationQuery = useQuery({
    queryKey: ["organization"],
    queryFn: () =>
      fetchInternalApi({
        target: "getOrganization",
        method: "GET",
      }),
    refetchOnWindowFocus: false,
    select: (data) => data.data,
  });

  const contractQuery = useQuery({
    queryKey: ["contract"],
    queryFn: () =>
      fetchInternalApi({
        target: "getContract",
        method: "GET",
      }),
    refetchOnWindowFocus: false,
    select: (data) => data.data,
  });

  const functionQuery = useQuery({
    queryKey: ["function"],
    queryFn: () =>
      fetchInternalApi({
        target: "getFunction",
        method: "GET",
      }),
    refetchOnWindowFocus: false,
    select: (data) => data.data,
  });

  const reporterQuery = useQuery({
    queryKey: ["reporter"],
    queryFn: () =>
      fetchInternalApi({
        target: "getReporter",
        method: "GET",
      }),
    refetchOnWindowFocus: false,
    select: (data) => data.data,
  });

  return (
    <DelegatorContainer>
      <TitleBase>Delegator</TitleBase>
      <TabContextProvider initTab={"organization"}>
        <TabList tabs={delegatorTabs} />
        {delegatorTabs.map(({ tabId, label }) => (
          <TabPanel tabId={tabId} key={tabId}>
            {(() => {
              switch (tabId) {
                case "organization":
                  return organizationQuery.isLoading ? (
                    <IsLoadingBase>
                      Loading... Please wait a moment
                    </IsLoadingBase>
                  ) : (
                    organizationQuery.data.map((item: any) => (
                      <DataListBase key={item.id}>
                        {JSON.stringify(item, null, 2)}
                      </DataListBase>
                    ))
                  );
                case "contract":
                  return contractQuery.isLoading ? (
                    <IsLoadingBase>
                      Loading... Please wait a moment
                    </IsLoadingBase>
                  ) : (
                    contractQuery.data.map((item: any) => (
                      <DataListBase key={item.id}>
                        {JSON.stringify(item, null, 2)}
                      </DataListBase>
                    ))
                  );
                case "function":
                  return functionQuery.isLoading ? (
                    <IsLoadingBase>
                      Loading... Please wait a moment
                    </IsLoadingBase>
                  ) : (
                    functionQuery.data.map((item: any) => (
                      <DataListBase key={item.id}>
                        {JSON.stringify(item, null, 2)}
                      </DataListBase>
                    ))
                  );
                case "reporter":
                  return reporterQuery.isLoading ? (
                    <IsLoadingBase>
                      Loading... Please wait a moment
                    </IsLoadingBase>
                  ) : (
                    reporterQuery.data.map((item: any) => (
                      <DataListBase key={item.id}>
                        {JSON.stringify(item, null, 2)}
                      </DataListBase>
                    ))
                  );
                default:
                  return null;
              }
            })()}
          </TabPanel>
        ))}
      </TabContextProvider>
    </DelegatorContainer>
  );
};

export default Delegator;
