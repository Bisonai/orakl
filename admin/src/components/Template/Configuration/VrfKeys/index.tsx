import { useApi } from "@/lib/useApi";
import { IvrfKeysProps } from "@/utils/types";
import BasicButton from "@/components/Common/BasicButton";
import { useDimmedPopupContext } from "@/hook/useDimmedPopupContext";
import { useState, useEffect } from "react";
import {
  HeaderBase,
  TitleBase,
  TableBase,
  TableLabel,
  TableData,
  Container,
  TableRow,
} from "./styled";
import { IsLoadingBase } from "../../BullMonitor/DetailTable/styled";

const VrfKeys = () => {
  const { configQuery, deleteMutation } = useApi({
    fetchEndpoint: "getVrfKeysConfig",
    deleteEndpoint: "modifyVrfKeysConfig",
    key: "vrfKeysConfig",
  });

  const { openDimmedPopup, closeDimmedPopup } = useDimmedPopupContext();
  const [localData, setLocalData] = useState<IvrfKeysProps[]>(
    configQuery.data || []
  );

  useEffect(() => {
    setLocalData(configQuery.data || []);
  }, [configQuery.data]);

  const handleDelete = async (id: string | number) => {
    await deleteMutation.mutateAsync(id);
    setLocalData((old) => old.filter((item) => item.id !== id));
  };

  return (
    <Container>
      <HeaderBase>
        <TitleBase>VRF Keys</TitleBase>
      </HeaderBase>
      {configQuery.isLoading ? (
        <IsLoadingBase>Loading... Please wait a moment</IsLoadingBase>
      ) : (
        localData.map((key) => (
          <TableBase key={key.id}>
            <BasicButton
              justifyContent="center"
              margin={"0 0 0 auto"}
              width="150px"
              text={"Remove"}
              onClick={() =>
                openDimmedPopup({
                  title: "Delete VRF Key",
                  confirmText: "Delete",
                  cancelText: "Cancel",
                  size: "small",
                  buttonTwo: true,
                  form: false,
                  onConfirm: () => {
                    handleDelete(key.id);
                    closeDimmedPopup();
                  },
                  onCancel: closeDimmedPopup,
                })
              }
            />
            <TableRow>
              <TableLabel>ID</TableLabel>
              <TableData>{key.id}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>SK</TableLabel>
              <TableData>{key.sk}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>PK</TableLabel>
              <TableData>{key.pk}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>PKX</TableLabel>
              <TableData>{key.pkX}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>PKY</TableLabel>
              <TableData>{key.pkY}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>Key Hash</TableLabel>
              <TableData>{key.keyHash}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>Chain</TableLabel>
              <TableData>{key.chain}</TableData>
            </TableRow>
          </TableBase>
        ))
      )}
    </Container>
  );
};

export default VrfKeys;
