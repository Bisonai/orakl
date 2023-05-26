import { fetchInternalApi } from "@/utils/api";
import { QueryFunctionContext, useQuery } from "react-query";
import {
  HeaderItem,
  QueueNameBase,
  TableContainer,
  TableDataContainer,
  TableHeaderContainer,
} from "./styled";
import { IQueueData } from "@/utils/types";
import Link from "next/link";

const MonitorTable = ({ serviceId }: { serviceId: string }) => {
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
    <TableContainer>
      <TableHeaderContainer>
        <QueueNameBase>QUEUE NAME</QueueNameBase>
        <HeaderItem>STATUS</HeaderItem>
        <HeaderItem>ACTIVE</HeaderItem>
        <HeaderItem>WAITING</HeaderItem>
        <HeaderItem>COMPLETED</HeaderItem>
        <HeaderItem>FAILED</HeaderItem>
        <HeaderItem>DELAYED</HeaderItem>
        <HeaderItem>PAUSED</HeaderItem>
      </TableHeaderContainer>
      {serviceQuery.data?.map((queue: IQueueData) => (
        <TableDataContainer key={queue.queue}>
          <Link href={`/bullmonitor/${serviceId}?queue=${queue.queue}`}>
            <QueueNameBase>{queue.queue}</QueueNameBase>
          </Link>
          <Link href={`/bullmonitor/${serviceId}?queue=${queue.queue}`}>
            <HeaderItem style={{ color: "white" }}>
              {queue.status ? "True" : "False"}
            </HeaderItem>
          </Link>

          <Link
            href={`/bullmonitor/${serviceId}?queue=${queue.queue}&activetab=active`}
          >
            <HeaderItem>{queue.active}</HeaderItem>
          </Link>
          <Link
            href={`/bullmonitor/${serviceId}?queue=${queue.queue}&activetab=waiting`}
          >
            <HeaderItem>{queue.waiting}</HeaderItem>
          </Link>
          <Link
            href={`/bullmonitor/${serviceId}?queue=${queue.queue}&activetab=completed`}
          >
            <HeaderItem>{queue.completed}</HeaderItem>
          </Link>
          <Link
            href={`/bullmonitor/${serviceId}?queue=${queue.queue}&activetab=failed`}
          >
            <HeaderItem
              style={{ color: queue.failed >= 1 ? "#ff5c5b" : "#00ADB5" }}
            >
              {queue.failed}
            </HeaderItem>
          </Link>
          <Link
            href={`/bullmonitor/${serviceId}?queue=${queue.queue}&activetab=delayed`}
          >
            <HeaderItem>{queue.delayed}</HeaderItem>
          </Link>
          <Link
            href={`/bullmonitor/${serviceId}?queue=${queue.queue}&activetab=paused`}
          >
            <HeaderItem>{queue.paused}</HeaderItem>
          </Link>
        </TableDataContainer>
      ))}
    </TableContainer>
  );
};

export default MonitorTable;
