import { useApi } from "@/lib/useApi";
import { IListenerProps } from "@/utils/types";
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
import { useState, useEffect } from "react";
import { IsLoadingBase } from "../../BullMonitor/DetailTable/styled";

const Listener = () => {
  const { configQuery, addMutation, deleteMutation } = useApi({
    fetchEndpoint: "getListenerConfig",
    deleteEndpoint: "modifyListenerConfig",
    key: "listenerConfig",
  });

  const { openDimmedPopup, closeDimmedPopup } = useDimmedPopupContext();
  const [localData, setLocalData] = useState<IListenerProps[]>(
    configQuery.data || []
  );

  useEffect(() => {
    setLocalData(configQuery.data || []);
  }, [configQuery.data]);

  const handleAdd = async (newData: string | undefined) => {
    if (!newData) {
      return;
    }

    const { data } = await addMutation.mutateAsync(newData);
    setLocalData((old) => [...old, data]);
  };

  const handleDelete = async (id: string | number) => {
    await deleteMutation.mutateAsync(id);
    setLocalData((old) => old.filter((item) => item.id !== id));
  };

  return (
    <Container>
      <HeaderBase>
        <TitleBase>Listeners</TitleBase>
      </HeaderBase>
      {configQuery.isLoading ? (
        <IsLoadingBase>Loading... Please wait a moment</IsLoadingBase>
      ) : (
        localData.map((listener, index) => (
          <TableBase key={listener.id}>
            <BasicButton
              justifyContent="center"
              margin={"0 0 0 auto"}
              width="150px"
              text={"Remove"}
              onClick={() =>
                openDimmedPopup({
                  title: "Delete Listener",
                  confirmText: "Delete",
                  cancelText: "Cancel",
                  size: "small",
                  buttonTwo: true,
                  form: false,
                  onConfirm: () => {
                    handleDelete(listener.id);
                    closeDimmedPopup();
                  },
                  onCancel: closeDimmedPopup,
                })
              }
            />
            <TableRow>
              <TableLabel>ID</TableLabel>
              <TableData>{listener.id}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>Address</TableLabel>
              <TableData>{listener.address}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>Event Name</TableLabel>
              <TableData>{listener.eventName}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>Service</TableLabel>
              <TableData>{listener.service}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>Chain</TableLabel>
              <TableData>{listener.chain}</TableData>
            </TableRow>
          </TableBase>
        ))
      )}
    </Container>
  );
};

export default Listener;
