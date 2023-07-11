import { useState } from "react";
import { useApi } from "@/lib/useApi";
import {
  TableBase,
  TableData,
  TitleBase,
  HeaderBase,
  TableLabel,
  TableRow,
  Container,
} from "./styled";
import BasicButton from "@/components/Common/BasicButton";
import { useDimmedPopupContext } from "@/hook/useDimmedPopupContext";
import {
  IsLoadingBase,
  NoDataAvailableBase,
} from "@/components/Template/BullMonitor/DetailTable/styled";

interface TableConfigProps {
  fetchEndpoint: string;
  deleteEndpoint: string;
  apiKey: string;
  title: string;
  dataLabels: string[];
  jsonData: {};
}

const DataTable = ({
  fetchEndpoint,
  deleteEndpoint,
  apiKey,
  title,
  dataLabels,
  jsonData,
}: TableConfigProps) => {
  const { configQuery, addMutation, deleteMutation } = useApi({
    fetchEndpoint,
    deleteEndpoint,
    key: apiKey,
  });

  const { openDimmedPopup, closeDimmedPopup } = useDimmedPopupContext();

  const [data, setData] = useState<any[]>(configQuery.data || []);

  const handleAdd = async (newData: any) => {
    try {
      await addMutation.mutateAsync(newData);
      setData((prevData) => [...prevData, newData]);
    } catch (error) {
      console.error("Error when adding:", error);
    }
  };

  const handleDelete = async (id: string | number) => {
    await deleteMutation.mutateAsync(id);
    setData((prevData) => prevData.filter((item) => item.id !== id));
  };

  const displayData = (data: any, label: string) => {
    if (label === "active") {
      return data ? "Active" : "Inactive";
    }
    return data;
  };

  console.log(data, "data");

  return (
    <Container>
      <HeaderBase>
        <TitleBase>{title}</TitleBase>
        <BasicButton
          height="50px"
          background="rgb(114, 250, 147)"
          justifyContent="center"
          margin={"0 30px 0 auto"}
          width="150px"
          text={"Add"}
          onClick={() =>
            openDimmedPopup({
              title: `Add ${title}`,
              confirmText: "Add",
              cancelText: "Cancel",
              size: "large",
              buttonTwo: true,
              jsonForm: jsonData,
              onConfirm: (newData: any) => {
                handleAdd(newData);
                closeDimmedPopup();
              },
              onCancel: closeDimmedPopup,
            })
          }
        />
      </HeaderBase>
      {configQuery.isLoading ? (
        <IsLoadingBase>Loading... Please wait a moment</IsLoadingBase>
      ) : data.length > 0 ? (
        data.map((item: any) => (
          <TableBase key={item.id}>
            <BasicButton
              justifyContent="center"
              margin={"0 0 0 auto"}
              width="150px"
              text={"Remove"}
              onClick={() =>
                openDimmedPopup({
                  title: `Delete ${title}`,
                  confirmText: "Delete",
                  cancelText: "Cancel",
                  size: "small",
                  buttonTwo: true,
                  onConfirm: () => {
                    handleDelete(item.id);
                    closeDimmedPopup();
                  },
                  onCancel: closeDimmedPopup,
                })
              }
            />
            {dataLabels.map((label) => (
              <TableRow key={label}>
                <TableLabel>{label}</TableLabel>
                <TableData>{displayData(item[label], label)}</TableData>
              </TableRow>
            ))}
          </TableBase>
        ))
      ) : (
        <NoDataAvailableBase>No data found</NoDataAvailableBase>
      )}
    </Container>
  );
};

export default DataTable;
