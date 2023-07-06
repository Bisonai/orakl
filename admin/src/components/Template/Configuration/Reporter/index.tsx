import { useApi } from "@/lib/useApi";
import { IReporterProps } from "@/utils/types";
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

const Reporter = () => {
  const { configQuery, deleteMutation } = useApi({
    fetchEndpoint: "getReporterConfig",
    deleteEndpoint: "modifyReporterConfig",
    key: "reporterConfig",
  });

  const { openDimmedPopup, closeDimmedPopup } = useDimmedPopupContext();
  const [localData, setLocalData] = useState<IReporterProps[]>(
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
        <TitleBase>Reporters</TitleBase>
      </HeaderBase>
      {configQuery.isLoading ? (
        <IsLoadingBase>Loading... Please wait a moment</IsLoadingBase>
      ) : (
        localData.map((reporter) => (
          <TableBase key={reporter.id}>
            <BasicButton
              justifyContent="center"
              margin={"0 0 0 auto"}
              width="150px"
              text={"Remove"}
              onClick={() =>
                openDimmedPopup({
                  title: "Delete Reporter",
                  confirmText: "Delete",
                  cancelText: "Cancel",
                  size: "small",
                  buttonTwo: true,
                  form: false,
                  onConfirm: () => {
                    handleDelete(reporter.id);
                    closeDimmedPopup();
                  },
                  onCancel: closeDimmedPopup,
                })
              }
            />
            <TableRow>
              <TableLabel>ID</TableLabel>
              <TableData>{reporter.id}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>Address</TableLabel>
              <TableData>{reporter.address}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>Private Key</TableLabel>
              <TableData>{reporter.privateKey}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>Oracle Address</TableLabel>
              <TableData>{reporter.oracleAddress}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>Service</TableLabel>
              <TableData>{reporter.service}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>Chain</TableLabel>
              <TableData>{reporter.chain}</TableData>
            </TableRow>
          </TableBase>
        ))
      )}
    </Container>
  );
};

export default Reporter;
