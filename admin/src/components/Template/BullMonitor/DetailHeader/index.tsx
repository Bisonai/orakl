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
  data: any;
}) => {
  const [selectedQueue, setSelectedQueue] = useState("");

  useEffect(() => {
    const currentPath = window.location.pathname;
    const selectedQueueFromPath = currentPath.split("/").pop();
    setSelectedQueue(selectedQueueFromPath || "");
  }, []);

  const handleQueueSelect = (queue: string) => {
    setSelectedQueue(queue);
  };

  return (
    <>
      <DetailHeaderContainer>
        <DetailHeaderBase>
          <BasicButton text={serviceId} width="auto" justifyContent="center" />
        </DetailHeaderBase>
        {data?.map((item: IQueueData) => (
          <Link
            key={item.queue}
            href={`/bullmonitor/${serviceId}/${item.queue}`}
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
