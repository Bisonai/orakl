import TabContextProvider from "@/components/Common/TabContextProvider";
import TabList from "@/components/Common/TabList";
import TabPanel from "@/components/Common/TabPanel";
import { statusTabs } from "@/utils/types";

const DetailTab = ({}: {}) => {
  return (
    <>
      <TabList tabs={statusTabs} />
      {statusTabs.map(({ tabId, label }) => (
        <TabPanel tabId={tabId} key={tabId}>
          {tabId}
        </TabPanel>
      ))}
    </>
  );
};

export default DetailTab;
