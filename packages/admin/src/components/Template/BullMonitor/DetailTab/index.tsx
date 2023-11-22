import TabContextProvider from "@/components/Common/TabContextProvider";
import TabList from "@/components/Common/TabList";
import TabPanel from "@/components/Common/TabPanel";
import { IQueueData, statusTabs } from "@/utils/types";
import DetailTable from "../DetailTable";
import { useQuery } from "react-query";
import { fetchInternalApi } from "@/utils/api";

const DetailTab = ({
  serviceId,
  data,
}: {
  serviceId: string;
  data: IQueueData[];
}) => {
  // These functions are only for displaying the number of data in each tab
  const currentQueue =
    typeof window !== "undefined"
      ? new URLSearchParams(window.location.search).get("queue")
      : null;
  const queueItem = data?.find((item) => item.queue === currentQueue);
  const queueName = queueItem?.queue;
  const queueStatusQuery = useQuery({
    queryKey: [
      "queueStatus",
      { serviceName: serviceId, queueName: queueName || "" },
    ],
    queryFn: () =>
      fetchInternalApi(
        {
          target: "queueStatus",
          method: "GET",
        },
        [{ serviceName: serviceId, queueName: queueName || "", status }]
      ),
    refetchOnWindowFocus: false,
    select: (statusData) => statusData.data,
  });
  const parsedNumberOfData = queueStatusQuery.data
    ? Object.values(queueStatusQuery.data).map(
        (numberOfJobs) => numberOfJobs as number
      )
    : undefined;

  return (
    <>
      <TabList tabs={statusTabs} numberOfData={parsedNumberOfData} />
      {statusTabs.map(({ tabId, label }) => (
        <TabPanel tabId={tabId} key={tabId}>
          <DetailTable serviceName={serviceId} status={tabId} data={data} />
        </TabPanel>
      ))}
    </>
  );
};

export default DetailTab;
