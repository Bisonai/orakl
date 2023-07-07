import { fetchInternalApi } from "@/utils/api";
import React from "react";
import { useQuery } from "react-query";
import TabContextProvider from "@/components/Common/TabContextProvider";
import TabList from "@/components/Common/TabList";
import TabPanel from "@/components/Common/TabPanel";
import { IQueueData, StatusTab, statusTabs } from "@/utils/types";

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

  console.log(
    organizationQuery.data,
    contractQuery.data,
    functionQuery.data,
    reporterQuery.data,
    "data"
  );

  const delegatorTabs: StatusTab[] = [
    { tabId: "organization", label: "Organization" },
    { tabId: "contract", label: "Contract" },
    { tabId: "function", label: "Function" },
    { tabId: "reporter", label: "Reporter" },
  ];

  return (
    <div>
      <h2 style={{ color: "white" }}>Delegator</h2>
      <TabContextProvider initTab={"organization"}>
        <TabList tabs={delegatorTabs} />
        {delegatorTabs.map(({ tabId, label }) => (
          <TabPanel tabId={tabId} key={tabId}>
            <div>
              {(() => {
                switch (tabId) {
                  case "organization":
                    return organizationQuery.isLoading ? (
                      <div>Loading organization data...</div>
                    ) : (
                      <pre>
                        {JSON.stringify(organizationQuery.data, null, 2)}
                      </pre>
                    );
                  case "contract":
                    return contractQuery.isLoading ? (
                      <div>Loading contract data...</div>
                    ) : (
                      <pre>{JSON.stringify(contractQuery.data, null, 2)}</pre>
                    );
                  case "function":
                    return functionQuery.isLoading ? (
                      <div>Loading function data...</div>
                    ) : (
                      <pre>{JSON.stringify(functionQuery.data, null, 2)}</pre>
                    );
                  case "reporter":
                    return reporterQuery.isLoading ? (
                      <div>Loading reporter data...</div>
                    ) : (
                      <pre>{JSON.stringify(reporterQuery.data, null, 2)}</pre>
                    );
                  default:
                    return null;
                }
              })()}
            </div>
          </TabPanel>
        ))}
      </TabContextProvider>
    </div>
  );
};

export default Delegator;
