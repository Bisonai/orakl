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
import { useEffect, useState } from "react";
import React from "react";
import { TablePagination } from "@mui/material";
import RefreshIcon from "@/components/Common/refreshIcon";
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
  const [page, setPage] = useState(1);
  const [rowsPerPage, setRowsPerPage] = React.useState(10);
  const { addToast } = useToastContext();
  const handleButtonClick = (text: any) => {
    setSelectedTab(text);
  };
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
          params: { limit: rowsPerPage, page },
        },
        [{ serviceName, queueName: queueName || "", status }]
      ),
    refetchOnWindowFocus: false,
    select: (statusData) => statusData.data,
  });
  console.log(selectedTab, "selectedTab");
  const handleRefresh = () => {
    statusQuery.refetch();
    addToast({
      type: ToastType.SUCCESS,
      title: "Refetched",
      content: `Successfully refetched ${serviceName} data`,
    });
  };

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
  const dataToDisplay = statusQuery.data?.slice(
    page * rowsPerPage,
    page * rowsPerPage + rowsPerPage
  );

  console.log("dataToDisplay", dataToDisplay, statusQuery);
  console.log("refreshing", statusQuery.isFetching);
  useEffect(() => {
    console.log(`Is fetching: ${statusQuery.isFetching}`);
  }, [statusQuery.isFetching]);

  return (
    <>
      {dataToDisplay?.length > 1 && (
        <DetailTableHeaderBase>
          <TablePagination
            component="div"
            count={statusQuery.data?.length || 0}
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
      {statusQuery.isLoading ? (
        <IsLoadingBase>Loading... Please wait a moment</IsLoadingBase>
      ) : dataToDisplay && dataToDisplay.length > 0 ? (
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
                    onClick={() => handleButtonClick("Data")}
                  />
                  <BasicButton
                    selected={selectedTab === "Option"}
                    width="auto"
                    margin="5px"
                    text="Option"
                    onClick={() => handleButtonClick("Option")}
                  />
                  <BasicButton
                    selected={selectedTab === "Logs"}
                    width="auto"
                    margin="5px"
                    text="Logs"
                    onClick={() => handleButtonClick("Logs")}
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
                  {selectedTab === "Data" && (
                    <CodeSnippetBase>
                      <pre>{JSON.stringify(item.data, null, 2)}</pre>
                    </CodeSnippetBase>
                  )}

                  {selectedTab === "Option" && (
                    <CodeSnippetBase>
                      <pre>{JSON.stringify(item.opts, null, 2)}</pre>
                    </CodeSnippetBase>
                  )}

                  {selectedTab === "Logs" &&
                    item.stacktrace &&
                    item.stacktrace.map((log: string, index: number) => (
                      <CodeSnippetBase key={index}>
                        <pre>{JSON.stringify(log, null, 2)}</pre>
                      </CodeSnippetBase>
                    ))}
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
