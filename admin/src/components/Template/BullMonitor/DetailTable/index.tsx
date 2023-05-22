import { fetchInternalApi } from "@/utils/api";
import { IQueueData } from "@/utils/types";
import { useQuery } from "react-query";

const DetailTable = ({
  serviceName,
  data,
  status,
}: {
  serviceName: string;
  data: IQueueData[];
  status: string;
}) => {
  let currentQueue: string | null = null;

  if (typeof window !== "undefined") {
    currentQueue = new URLSearchParams(window.location.search).get("queue");
  }
  const queueItem = data?.find((item) => item.queue === currentQueue);
  const queueName = queueItem?.queue;

  const statusQuery = useQuery({
    queryKey: [
      "queueStatus",
      { serviceName, queueName: queueName || "", status },
    ],
    queryFn: () =>
      fetchInternalApi(
        {
          target: "queueStatus",
          method: "GET",
        },
        [{ serviceName, queueName: queueName || "", status }]
      ),
    refetchOnWindowFocus: false,
    select: (statusData) => statusData.data,
  });

  return (
    <>
      <div style={{ margin: "0px 40px" }}>
        <div>{serviceName}</div>
        <div>{queueName}</div>
        <div>
          {JSON.stringify(
            statusQuery?.data && statusQuery?.data.length > 0
              ? statusQuery?.data[0]
              : null
          )}
        </div>
      </div>
    </>
  );
};

export default DetailTable;
