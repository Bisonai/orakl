import { useState } from "react";
import { useApi } from "@/lib/useApi";
import DataTable from "../ConfigurationTable/DataTable";
import BasicButton from "@/components/Common/BasicButton";

import {
  TitleBase,
  TableBase,
  TableLabel,
  TableData,
  Container,
  TableRow,
  AddDataBase,
  AddDataFormBase,
  AddDataForm,
} from "./styled";
import { useDimmedPopupContext } from "@/hook/useDimmedPopupContext";

const Adapter = () => {
  const { openDimmedPopup, closeDimmedPopup } = useDimmedPopupContext();

  const { configQuery, addMutation, deleteMutation } = useApi({
    fetchEndpoint: "getAdapterConfig",
    deleteEndpoint: "modifyAdapterConfig",
    key: "adapterConfig",
  });

  const [showForm, setShowForm] = useState(false);
  const [formData, setFormData] = useState({
    adapterHash: "",
    name: "",
    decimals: 0,
    feeds: [
      {
        name: "",
        definition: {
          url: "",
          headers: {
            "Content-Type": "application/json",
          },
          method: "GET",
          reducers: [
            {
              function: "PARSE",
              args: ["price"],
            },
            {
              function: "POW10",
              args: 8,
            },
            {
              function: "ROUND",
            },
          ],
        },
      },
    ],
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
      // parsedValue = {
      //   adapterHash: "",
      //   name: "",
      //   decimals: 0,
      //   feeds: [
      //     {
      //       name: "",
      //       definition: {
      //         url: "",
      //         headers: {
      //           "Content-Type": "application/json",
      //         },
      //         method: "GET",
      //         reducers: [
      //           {
      //             function: "PARSE",
      //             args: ["price"],
      //           },
      //           {
      //             function: "POW10",
      //             args: 8,
      //           },
      //           {
      //             function: "ROUND",
      //           },
      //         ],
      //       },
      //     },
      //   ],
      // };
      console.error("Error parsing JSON:", error);
    }

    setFormData((prevData) => ({
      ...prevData,
      ...parsedValue,
    }));
  };

  const handleAddClick = () => {
    fetch(`${process.env.NEXT_PUBLIC_API_BASE_URL}/api/v1/adapter`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Accept: "*/*",
      },
      body: JSON.stringify(formData),
    })
      .then((response) => response.json())
      .then((data) => {
        console.log("Success:", data);
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
      <TitleBase>Adapter</TitleBase>
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
              margin="0 auto"
              width="150px"
              text="Submit"
              onClick={handleAddClick}
            />
          </AddDataFormBase>
        )}
      </AddDataBase>
      {configQuery.isLoading ? (
        <div>Loading... Please wait a moment</div>
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
                  title: `Delete Adapter`,
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

export default Adapter;
