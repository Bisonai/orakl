"use client";

import { fetchInternalApi } from "@/utils/api";
import { useQuery } from "react-query";
import DetailHeader from "./DetailHeader";
import DetailTab from "./DetailTab";
import TabContextProvider from "@/components/Common/TabContextProvider";

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
      <TabContextProvider initTab={"active"}>
        <div style={{ background: "#222831;", margin: "0px 40px" }}>
          <DetailHeader serviceId={serviceId} data={serviceQuery?.data} />
          <DetailTab serviceId={serviceId} data={serviceQuery?.data} />
        </div>
      </TabContextProvider>
    </>
  );
}
