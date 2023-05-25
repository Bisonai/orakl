import * as React from "react";

import VrfTable from "./VrfTable";
import RequestResponseTable from "./RequestResponseTable";
import AggregatorTable from "./AggregatorTable";
import { useQuery } from "react-query";
import { fetchInternalApi } from "@/utils/api";
import { IQueueInfoData } from "@/utils/types";

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

  const vrfData = queuesInfoQuery.data?.find(
    (data: IQueueInfoData) => data.serviceName === "vrf"
  );
  const requestResponseData = queuesInfoQuery.data?.find(
    (data: IQueueInfoData) => data.serviceName === "request-response"
  );
  const aggregatorData = queuesInfoQuery.data?.find(
    (data: IQueueInfoData) => data.serviceName === "aggregator"
  );

  return (
    <>
      <div style={{ maxWidth: "1400px" }}>
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
