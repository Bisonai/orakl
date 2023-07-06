import { useApi } from "@/lib/useApi";
import { IAdapterProps } from "@/utils/types";
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
import { IsLoadingBase } from "../../BullMonitor/DetailTable/styled";

const Adapter = () => {
  const { configQuery, deleteMutation } = useApi({
    fetchEndpoint: "getAdapterConfig",
    deleteEndpoint: "modifyAdapterConfig",
    key: "adapterConfig",
  });

  const { openDimmedPopup, closeDimmedPopup } = useDimmedPopupContext();
  const data: IAdapterProps[] = configQuery.data || [];

  const handleDelete = async (id: string | number) => {
    await deleteMutation.mutateAsync(id);
  };

  return (
    <Container>
      <HeaderBase>
        <TitleBase>Adapters</TitleBase>
      </HeaderBase>
      {configQuery.isLoading ? (
        <IsLoadingBase>Loading... Please wait a moment</IsLoadingBase>
      ) : (
        data.map((adapter) => (
          <TableBase key={adapter.id}>
            <BasicButton
              justifyContent="center"
              margin={"0 0 0 auto"}
              width="150px"
              text={"Remove"}
              onClick={() =>
                openDimmedPopup({
                  title: "Delete Adapter",
                  confirmText: "Delete",
                  cancelText: "Cancel",
                  size: "small",
                  buttonTwo: true,
                  form: false,
                  onConfirm: () => {
                    handleDelete(adapter.id);
                    closeDimmedPopup();
                  },
                  onCancel: closeDimmedPopup,
                })
              }
            />
            <TableRow>
              <TableLabel>ID</TableLabel>
              <TableData>{adapter.id}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>Adapter Hash</TableLabel>
              <TableData>{adapter.adapterHash}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>Name</TableLabel>
              <TableData>{adapter.name}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>Decimals</TableLabel>
              <TableData>{adapter.decimals}</TableData>
            </TableRow>
          </TableBase>
        ))
      )}
    </Container>
  );
};

export default Adapter;
