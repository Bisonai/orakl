import React, { useState } from "react";
import { fetchInternalApi } from "@/utils/api";
import { IQueueData, ToastType } from "@/utils/types";
import { useQuery } from "react-query";
import {
  DetailTableContainer,
  DetailHeaderBase,
  DetailTableBase,
  DetailLeftBase,
  DetailRightBase,
  TimeTableTextBase,
  DetailTabBase,
  CodeSnippetBase,
  ServiceNameBase,
  JobIdBase,
  NoDataAvailableBase,
  IsLoadingBase,
  DetailTableHeaderBase,
} from "./styled";
import BasicButton from "@/components/Common/BasicButton";
import { TablePagination } from "@mui/material";
import { useToastContext } from "@/hook/useToastContext";
import { StyledButton } from "@/theme/theme";

const DetailTable = ({
  serviceName,
  data,
  status,
}: {
  serviceName: string;
  data: IQueueData[];
  status: string;
}) => {
  const [selectedTab, setSelectedTab] = useState("Data");
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(10);
  const { addToast } = useToastContext();

  const handleChangePage = (
    event: React.MouseEvent<HTMLButtonElement> | null,
    newPage: number
  ) => {
    setPage(newPage);
  };

  const handleChangeRowsPerPage = (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>
  ) => {
    setRowsPerPage(parseInt(event.target.value, 10));
    setPage(0);
  };

  const currentQueue =
    typeof window !== "undefined"
      ? new URLSearchParams(window.location.search).get("queue")
      : null;

  const queueItem = data?.find((item) => item.queue === currentQueue);
  const queueName = queueItem?.queue;

  const queueStatusQuery = useQuery({
    queryKey: [
      "queueStatus",
      { serviceName, queueName: queueName || "", status },
    ],
    queryFn: () =>
      fetchInternalApi(
        {
          target: "queueStatus",
          method: "GET",
          params: { limit: rowsPerPage, page },
        },
        [{ serviceName, queueName: queueName || "", status }]
      ),
    refetchOnWindowFocus: false,
    select: (statusData) => statusData.data,
  });

  const handleRefresh = () => {
    queueStatusQuery.refetch();
    addToast({
      type: ToastType.SUCCESS,
      title: "Refetched",
      content: `Successfully refetched ${serviceName} data`,
    });
  };

  const handleTabChange = (newTab: string) => setSelectedTab(newTab);

  const renderTabContent = (item: any, tab: string) => {
    switch (tab) {
      case "Data":
        return <pre>{JSON.stringify(item.data, null, 2)}</pre>;
      case "Option":
        return <pre>{JSON.stringify(item.opts, null, 2)}</pre>;
      case "Logs":
        return item.stacktrace && <pre>{item.stacktrace.join("\n")}</pre>;
      default:
        return null;
    }
  };

  const dataToDisplay = queueStatusQuery.data?.slice(
    page * rowsPerPage,
    page * rowsPerPage + rowsPerPage
  );

  return (
    <>
      {queueStatusQuery?.data?.length >= 1 && (
        <DetailTableHeaderBase>
          <TablePagination
            component="div"
            count={queueStatusQuery.data?.length || 0}
            page={page}
            onPageChange={handleChangePage}
            rowsPerPage={rowsPerPage}
            onRowsPerPageChange={handleChangeRowsPerPage}
            color="secondary"
            sx={{
              ".MuiTablePagination-displayedRows": {
                color: "#02c7d1",
              },
              ".MuiTablePagination-selectLabel": {
                color: "#02c7d1",
              },
              ".MuiTablePagination-select": { color: "#02c7d1" },
              button: { color: "#02c7d1" },
            }}
          />
          <StyledButton
            onClick={handleRefresh}
            size="large"
            color="secondary"
            variant="contained"
            style={{ height: "40px", width: "100px", marginTop: "6px" }}
          >
            Refetch
          </StyledButton>
        </DetailTableHeaderBase>
      )}
      {queueStatusQuery.isLoading ? (
        <IsLoadingBase>Loading... Please wait a moment</IsLoadingBase>
      ) : queueStatusQuery?.data && queueStatusQuery?.data?.length >= 1 ? (
        dataToDisplay.map((item: any, index: number) => (
          <div style={{ width: "100%" }} key={index}>
            <DetailTableContainer>
              <DetailHeaderBase>
                <div>
                  <ServiceNameBase>{serviceName.toUpperCase()}</ServiceNameBase>
                  <JobIdBase>{item.opts.jobId}</JobIdBase>
                </div>
                <DetailTabBase>
                  <BasicButton
                    selected={selectedTab === "Data"}
                    width="auto"
                    margin="5px"
                    text="Data"
                    onClick={() => handleTabChange("Data")}
                  />
                  <BasicButton
                    selected={selectedTab === "Option"}
                    width="auto"
                    margin="5px"
                    text="Option"
                    onClick={() => handleTabChange("Option")}
                  />
                  <BasicButton
                    selected={selectedTab === "Logs"}
                    width="auto"
                    margin="5px"
                    text="Logs"
                    onClick={() => handleTabChange("Logs")}
                  />
                </DetailTabBase>
              </DetailHeaderBase>
              <DetailTableBase>
                <DetailLeftBase>
                  <TimeTableTextBase>
                    Added at {new Date(item.timestamp).toLocaleString("en-US")}
                  </TimeTableTextBase>
                  <TimeTableTextBase>
                    Processed at{" "}
                    {new Date(item.processedOn).toLocaleString("en-US")}
                  </TimeTableTextBase>
                  <TimeTableTextBase>
                    Finished at{" "}
                    {new Date(item.finishedOn).toLocaleString("en-US")}
                  </TimeTableTextBase>
                </DetailLeftBase>
                <DetailRightBase>
                  <CodeSnippetBase>
                    {renderTabContent(item, selectedTab)}
                  </CodeSnippetBase>
                </DetailRightBase>
              </DetailTableBase>
            </DetailTableContainer>
          </div>
        ))
      ) : (
        <NoDataAvailableBase>No data available</NoDataAvailableBase>
      )}
    </>
  );
};

export default DetailTable;
