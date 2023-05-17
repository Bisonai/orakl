import { useState } from "react";
import MonitorTable from "../MonitorTable";
import TableHeader from "../TableHeader";

const AggregatorTable = ({ serviceId }: { serviceId: string }): JSX.Element => {
  const [refreshKey, setRefreshKey] = useState(0); // 새로고침 키 상태값

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
        buttonText="Aggregator"
        onRefresh={handleRefresh}
      />
      <MonitorTable serviceId={serviceId} key={refreshKey} />
    </>
  );
};

export default AggregatorTable;
