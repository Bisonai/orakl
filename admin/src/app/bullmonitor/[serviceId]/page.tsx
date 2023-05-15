"use client";

import BullMonitorDetailTemplate from "@/components/Template/BullMonitor/detail";

export default function ServiceDetail({
  params,
}: {
  params: { serviceId: string };
}) {
  return <BullMonitorDetailTemplate serviceId={params.serviceId} />;
}
