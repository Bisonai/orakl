import { useState } from "react";
import MonitorTable from "../MonitorTable";
import TableHeader from "../TableHeader";

const RequestResponseTable = ({ serviceId }: { serviceId: string }) => {
  const [refreshKey, setRefreshKey] = useState(0);

  const handleRefresh = () => {
    setRefreshKey((prevKey) => prevKey + 1);
  };

  return (
    <>
      <TableHeader
        version={""}
        memoryUsage={""}
        fragmentationRatio={""}
        connectedClients={""}
        blockedClients={""}
        buttonText="Request-Response"
        onRefresh={handleRefresh}
      />
      <MonitorTable serviceId={serviceId} key={refreshKey} />{" "}
    </>
  );
};

export default RequestResponseTable;
