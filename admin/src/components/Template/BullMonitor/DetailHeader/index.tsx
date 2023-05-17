import { useState, useEffect } from "react";
import BasicButton from "@/components/Common/BasicButton";
import { fetchInternalApi } from "@/utils/api";
import { useQuery } from "react-query";
import { DetailHeaderBase, DetailHeaderContainer } from "./styled";
import RefreshIcon from "@/components/Common/refreshIcon";
import { IQueueData } from "@/utils/types";
import Link from "next/link";

const DetailHeader = ({
  serviceId,
  data,
}: {
  serviceId: string;
  data: IQueueData[];
}) => {
  const [selectedQueue, setSelectedQueue] = useState("");

  useEffect(() => {
    const url = new URL(window.location.href);
    const queue = url.searchParams.get("queue");
    if (queue) {
      setSelectedQueue(queue);
    }
  }, []);

  useEffect(() => {
    const url = new URL(window.location.href);
    const queue = url.searchParams.get("queue");

    if (data && data?.length > 0 && !queue && selectedQueue === "") {
      const newSelectedQueue = data?.sort((a, b) =>
        a.queue.localeCompare(b.queue)
      )?.[0].queue;
      const url = new URL(window.location.href);
      url.searchParams.set("queue", newSelectedQueue);
      window.history.replaceState({}, "", url.toString());

      setSelectedQueue(newSelectedQueue);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data]);

  const handleQueueSelect = (queue: string) => {
    setSelectedQueue(queue);
  };

  return (
    <>
      <DetailHeaderContainer>
        <DetailHeaderBase>
          <BasicButton text={serviceId} width="auto" justifyContent="center" />
        </DetailHeaderBase>
        {data
          ?.sort((a, b) => a.queue.localeCompare(b.queue))
          ?.map((item) => (
            <Link
              key={item.queue}
              href={`/bullmonitor/${serviceId}?queue=${item.queue}`}
              replace={true}
            >
              <BasicButton
                text={item.queue}
                width="auto"
                margin="5px 10px 5px 0px"
                selected={selectedQueue === item.queue}
                onClick={() => handleQueueSelect(item.queue)}
              />
            </Link>
          ))}
      </DetailHeaderContainer>
    </>
  );
};

export default DetailHeader;
