"use client";

import { fetchInternalApi } from "@/utils/api";
import { useQuery } from "react-query";
import DetailHeader from "./DetailHeader";
import DetailTab from "./DetailTab";

export default function BullMonitorDetailTemplate({
  serviceId,
}: {
  serviceId: string;
}): JSX.Element {
  const serviceQuery = useQuery({
    queryKey: ["service", serviceId],
    queryFn: () =>
      fetchInternalApi(
        {
          target: "service",
          method: "GET",
        },
        [serviceId]
      ),
    refetchOnWindowFocus: false,
    select: (data) => data.data,
  });

  return (
    <>
      <div
        style={{
          background: "#222831",
          margin: "0px 40px",
          maxWidth: "1400px",
        }}
      >
        <DetailHeader serviceId={serviceId} data={serviceQuery?.data} />
        <DetailTab serviceId={serviceId} data={serviceQuery?.data} />
      </div>
    </>
  );
}
