import * as React from "react";

import VrfTable from "./VrfTable";
import RequestResponseTable from "./RequestResponseTable";
import AggregatorTable from "./AggregatorTable";

export default function BullMonitorTemplate() {
  return (
    <>
      <VrfTable serviceId={"vrf"} />
      <RequestResponseTable serviceId={"request-response"} />
      <AggregatorTable serviceId={"aggregator"} />
    </>
  );
}
