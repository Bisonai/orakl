import { fetchInternalApi } from "@/utils/api";
import { QueryFunctionContext, useQuery } from "react-query";
import {
  QueueNameBase,
  TableContainer,
  TableDataContainer,
  TableHeaderContainer,
} from "./styled";
import { IQueueData } from "@/utils/types";

const MonitorTable = ({ serviceId }: { serviceId: string }) => {
  const accountQuery = useQuery({
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

  console.log(accountQuery.data, accountQuery);
  return (
    <TableContainer>
      <TableHeaderContainer>
        <div style={{ minWidth: "300px", textAlign: "left" }}>QUEUE NAME</div>
        <div>STATUS</div>
        <div>ACTIVE</div>
        <div>WAITING</div>
        <div>COMPLETED</div>
        <div>FAILED</div>
        <div>DELAYED</div>
        <div>PAUSED</div>
      </TableHeaderContainer>
      {accountQuery.data?.map((queue: IQueueData) => (
        <TableDataContainer key={queue.queue}>
          <QueueNameBase>{queue.queue}</QueueNameBase>
          <div style={{ color: "white" }}>{accountQuery.status}</div>
          <div>{queue.active}</div>
          <div>{queue.waiting}</div>
          <div>{queue.completed}</div>
          <div style={{ color: queue.failed >= 1 ? "#ff5c5b" : "#49a7ff" }}>
            {queue.failed}
          </div>
          <div>{queue.delayed}</div>
          <div>{queue.paused}</div>
        </TableDataContainer>
      ))}
    </TableContainer>
  );
};

export default MonitorTable;
