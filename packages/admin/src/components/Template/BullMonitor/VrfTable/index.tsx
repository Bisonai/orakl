import { useState } from "react";
import MonitorTable from "../MonitorTable";
import TableHeader from "../TableHeader";
import { useToastContext } from "@/hook/useToastContext";
import { IQueueInfoData, ToastType } from "@/utils/types";

const VrfTable = ({
  serviceData,
  serviceId,
}: {
  serviceData: IQueueInfoData;
  serviceId: string;
}) => {
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
      <TableHeader serviceData={serviceData} onRefresh={handleRefresh} />
      <MonitorTable serviceId={serviceId} key={refreshKey} />
    </>
  );
};

export default VrfTable;
