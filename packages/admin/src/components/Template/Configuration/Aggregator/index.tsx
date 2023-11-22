import { useState } from "react";
import { useApi } from "@/lib/useApi";
import DataTable from "../ConfigurationTable/DataTable";
import BasicButton from "@/components/Common/BasicButton";

import { useDimmedPopupContext } from "@/hook/useDimmedPopupContext";
import {
  ErrorMessageBase,
  IsLoadingBase,
  NoDataAvailableBase,
} from "../../BullMonitor/DetailTable/styled";
import { useToastContext } from "@/hook/useToastContext";
import { ToastType } from "@/utils/types";
import {
  AddDataBase,
  AddDataForm,
  AddDataFormBase,
  Container,
  TableBase,
  TableData,
  TableLabel,
  TableRow,
  TitleBase,
} from "../Adapter/styled";

const Aggregator = () => {
  const { openDimmedPopup, closeDimmedPopup } = useDimmedPopupContext();
  const { addToast } = useToastContext();
  const { configQuery, addMutation, deleteMutation } = useApi({
    fetchEndpoint: "getAggregatorConfig",
    deleteEndpoint: "modifyAggregatorConfig",
    key: "AggregatorConfig",
  });

  const [showForm, setShowForm] = useState(false);
  const [formData, setFormData] = useState({
    aggregatorHash: "string",
    active: true,
    name: "string",
    address: "string",
    heartbeat: 0,
    threshold: 0.05,
    absoluteThreshold: 0.1,
    adapterHash: "string",
    chain: "string",
  });

  const handleFormChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    let parsedValue: any;

    try {
      parsedValue = JSON.parse(e.target.value);
      if (parsedValue.feeds && parsedValue.feeds.length > 0) {
        const feed = parsedValue.feeds[0];
        feed.definition = feed.definition || {};
      }
    } catch (error) {
      console.error("Error parsing JSON:", error);
    }

    setFormData((prevData) => ({
      ...prevData,
      ...parsedValue,
    }));
  };

  const handleAddClick = () => {
    fetch(`${process.env.NEXT_PUBLIC_API_BASE_URL}/api/v1/aggregator`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Accept: "*/*",
      },
      body: JSON.stringify(formData),
    })
      .then((response) => response.json())
      .then((data) => {
        if (data.status === 400) {
          console.log("error:", data.message);
          addToast({
            type: ToastType.ERROR,
            title: "ERROR",
            content: data.message,
          });
        }
      })
      .catch((error) => {
        console.error("Error:", error);
      });
  };
  const handleDelete = async (id: string | number) => {
    await deleteMutation.mutateAsync(id);
  };
  const toggleForm = () => {
    setShowForm((prevShowForm) => !prevShowForm);
  };

  const data = configQuery.data || [];

  const displayData = (data: any, label: string) => {
    if (label === "active") {
      return data ? "Active" : "Inactive";
    }
    return data;
  };

  return (
    <Container>
      <TitleBase>Aggregator</TitleBase>
      <AddDataBase>
        <BasicButton
          height="50px"
          background="rgb(114, 250, 147)"
          justifyContent="center"
          margin="0 0 40px auto"
          width="150px"
          text="Add"
          onClick={toggleForm}
        />
        {showForm && (
          <AddDataFormBase>
            <AddDataForm
              name="formData"
              rows={10}
              cols={50}
              value={JSON.stringify(formData, null, 2)}
              onChange={handleFormChange}
            />
            <BasicButton
              height="50px"
              background="rgb(114, 250, 147)"
              justifyContent="center"
              margin="0 0 0 auto"
              width="150px"
              text="Submit"
              onClick={handleAddClick}
            />
          </AddDataFormBase>
        )}
      </AddDataBase>
      {configQuery.isLoading ? (
        <IsLoadingBase>Loading... Please wait a moment</IsLoadingBase>
      ) : configQuery.isError ? (
        <ErrorMessageBase>Error occurred while fetching data</ErrorMessageBase>
      ) : data.length === 0 ? (
        <NoDataAvailableBase>No data found</NoDataAvailableBase>
      ) : (
        data.map((item: any) => (
          <TableBase key={item.id}>
            <BasicButton
              justifyContent="center"
              margin="0 0 0 auto"
              width="150px"
              text="Remove"
              onClick={() =>
                openDimmedPopup({
                  title: `Delete Aggregator`,
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
            {Object.keys(item).map((label) => (
              <TableRow key={label}>
                <TableLabel>{label}</TableLabel>
                <TableData>{displayData(item[label], label)}</TableData>
              </TableRow>
            ))}
          </TableBase>
        ))
      )}
    </Container>
  );
};

export default Aggregator;
