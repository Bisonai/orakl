import { useState } from "react";
import MonitorTable from "../MonitorTable";
import TableHeader from "../TableHeader";
import { useToastContext } from "@/hook/useToastContext";
import { ToastType } from "@/utils/types";

const AggregatorTable = ({ serviceId }: { serviceId: string }): JSX.Element => {
  const [refreshKey, setRefreshKey] = useState(0);
  const { addToast } = useToastContext();
  const handleRefresh = () => {
    setRefreshKey((prevKey) => prevKey + 1);
    addToast({
      type: ToastType.SUCCESS,
      title: "Refetched",
      content: `Successfully refetched Aggregator data`,
    });
  };

  return (
    <>
      <TableHeader
        version={""}
        memoryUsage={""}
        fragmentationRatio={""}
        connectedClients={""}
        blockedClients={""}
        buttonText="Aggregator"
        onRefresh={handleRefresh}
      />
      <MonitorTable serviceId={serviceId} key={refreshKey} />
    </>
  );
};

export default AggregatorTable;
