import * as React from "react";
import { useQuery } from "react-query";
import { fetchInternalApi } from "@/utils/api";

import { IQueueInfoData } from "@/utils/types";

import VrfTable from "./VrfTable";
import RequestResponseTable from "./RequestResponseTable";
import AggregatorTable from "./AggregatorTable";
import { removeCookie } from "@/lib/cookies";

export default function BullMonitorTemplate() {
  const queuesInfoQuery = useQuery({
    queryKey: ["queuesInfo"],
    queryFn: () =>
      fetchInternalApi(
        {
          target: "queuesInfo",
          method: "GET",
        },
        []
      ),
    refetchOnWindowFocus: false,
    select: (data) => data.data,
  });

  const handleErrorResponse = (error: any) => {
    if (error?.response?.status === 401) {
      removeCookie("token");
    }
  };

  const vrfData = queuesInfoQuery.data?.find(
    (data: IQueueInfoData) => data.serviceName === "vrf"
  );
  const requestResponseData = queuesInfoQuery.data?.find(
    (data: IQueueInfoData) => data.serviceName === "request-response"
  );
  const aggregatorData = queuesInfoQuery.data?.find(
    (data: IQueueInfoData) => data.serviceName === "aggregator"
  );

  if (queuesInfoQuery.isError) {
    handleErrorResponse(queuesInfoQuery.error);
  }

  return (
    <>
      <div style={{ width: "90%", paddingBottom: "100px" }}>
        <VrfTable serviceData={vrfData} serviceId={"vrf"} />
        <RequestResponseTable
          serviceData={requestResponseData}
          serviceId={"request-response"}
        />
        <AggregatorTable
          serviceData={aggregatorData}
          serviceId={"aggregator"}
        />
      </div>
    </>
  );
}
