import { useState } from "react";
import MonitorTable from "../MonitorTable";
import TableHeader from "../TableHeader";
import { useToastContext } from "@/hook/useToastContext";
import { ToastType } from "@/utils/types";

const VrfTable = ({ serviceId }: { serviceId: string }) => {
  const [refreshKey, setRefreshKey] = useState(0);
  const { addToast } = useToastContext();
  const handleRefresh = () => {
    setRefreshKey((prevKey) => prevKey + 1);
    addToast({
      type: ToastType.SUCCESS,
      title: "Refetched",
      content: `Successfully refetched VRF data`,
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
        buttonText="VRF"
        onRefresh={handleRefresh}
      />
      <MonitorTable serviceId={serviceId} key={refreshKey} />
    </>
  );
};

export default VrfTable;
