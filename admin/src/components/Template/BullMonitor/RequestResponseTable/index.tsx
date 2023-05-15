import { useState } from "react";
import MonitorTable from "../MonitorTable";
import TableHeader from "../TableHeader";

const RequestResponseTable = ({ serviceId }: { serviceId: string }) => {
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
        buttonText="Request Response"
        onRefresh={handleRefresh} // handleRefresh 함수를 onRefresh prop으로 전달
      />
      <MonitorTable serviceId={serviceId} key={refreshKey} />{" "}
      {/* key prop 추가 */}
    </>
  );
};

export default RequestResponseTable;
