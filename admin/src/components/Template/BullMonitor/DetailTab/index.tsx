import TabContextProvider from "@/components/Common/TabContextProvider";
import TabList from "@/components/Common/TabList";
import TabPanel from "@/components/Common/TabPanel";
import { IQueueData, statusTabs } from "@/utils/types";
import DetailTable from "../DetailTable";

const DetailTab = ({
  serviceId,
  data,
}: {
  serviceId: string;
  data: IQueueData[];
}) => {
  return (
    <>
      <TabList tabs={statusTabs} />
      {statusTabs.map(({ tabId, label }) => (
        <TabPanel tabId={tabId} key={tabId}>
          <DetailTable serviceName={serviceId} status={tabId} data={data} />
        </TabPanel>
      ))}
    </>
  );
};

export default DetailTab;
