import * as React from "react";
import TableHeader from "./TableHeader";
import MonitorTable from "./MonitorTable";
import VrfTable from "./VrfTable";
import RequestResponseTable from "./RequestResponseTable";
import AggregatorTable from "./AggregatorTable";

const version = "5.0.8";
const memoryUsage = "0.05%";
const fragmentationRatio = "1.39";
const connectedClients = "21";
const blockedClients = "5";
export default function BullMonitor(): JSX.Element {
  return (
    <>
      <VrfTable serviceId={"vrf"} />
      <RequestResponseTable serviceId={"request-response"} />
      <AggregatorTable serviceId={"aggregator"} />
    </>
  );
}
