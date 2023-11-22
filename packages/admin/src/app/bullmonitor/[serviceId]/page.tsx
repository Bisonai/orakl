"use client";

import BullMonitorDetailTemplate from "@/components/Template/BullMonitor/detail";
import { Route } from "@/utils/route";

export default function AccountDetail({
  params,
}: {
  params: { serviceId: string };
}) {
  let Template;
  if (
    [Route.vrf, Route.aggregator, Route["request-response"]].includes(
      params.serviceId as Route
    )
  ) {
    Template = BullMonitorDetailTemplate;
  } else if (Route.settings === params.serviceId) {
  } else if (Route.fetcher === params.serviceId) {
  } else {
    // TODO: 404
  }

  return Template ? <Template serviceId={params.serviceId} /> : null;
}
