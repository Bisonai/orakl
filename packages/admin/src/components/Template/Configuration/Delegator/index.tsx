import { fetchInternalApi } from "@/utils/api";
import React from "react";
import { useQuery } from "react-query";
import TabContextProvider from "@/components/Common/TabContextProvider";
import TabList from "@/components/Common/TabList";
import TabPanel from "@/components/Common/TabPanel";
import { StatusTab, delegatorTabs } from "@/utils/types";
import { DataListBase, DelegatorContainer, TitleBase } from "./styled";
import {
  IsLoadingBase,
  ErrorMessageBase,
  NoDataAvailableBase,
} from "@/components/Template/BullMonitor/DetailTable/styled";

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
              const query =
                tabId === "organization"
                  ? organizationQuery
                  : tabId === "contract"
                  ? contractQuery
                  : tabId === "function"
                  ? functionQuery
                  : tabId === "reporter"
                  ? reporterQuery
                  : null;

              if (query?.isLoading) {
                return (
                  <IsLoadingBase>Loading... Please wait a moment</IsLoadingBase>
                );
              }

              if (query?.isError) {
                return (
                  <ErrorMessageBase>
                    Error occurred while fetching data
                  </ErrorMessageBase>
                );
              }
              if (query?.data?.length === 0) {
                return <NoDataAvailableBase>No data found</NoDataAvailableBase>;
              }
              return query?.data?.map((item: any) => (
                <DataListBase key={item.id}>
                  {JSON.stringify(item, null, 2)}
                </DataListBase>
              ));
            })()}
          </TabPanel>
        ))}
      </TabContextProvider>
    </DelegatorContainer>
  );
};

export default Delegator;
