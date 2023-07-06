import { useApi } from "@/lib/useApi";
import { IAggregatorProps } from "@/utils/types";
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

const Aggregator = () => {
  const { configQuery, deleteMutation } = useApi({
    fetchEndpoint: "getAggregatorConfig",
    deleteEndpoint: "modifyAggregatorConfig",
    key: "aggregatorConfig",
  });

  const { openDimmedPopup, closeDimmedPopup } = useDimmedPopupContext();
  const data: IAggregatorProps[] = configQuery.data || [];

  const handleDelete = async (hash: string) => {
    await deleteMutation.mutateAsync(hash);
  };

  return (
    <Container>
      <HeaderBase>
        <TitleBase>Aggregators</TitleBase>
      </HeaderBase>
      {configQuery.isLoading ? (
        <IsLoadingBase>Loading... Please wait a moment</IsLoadingBase>
      ) : (
        data.map((aggregator) => (
          <TableBase key={aggregator.aggregatorHash}>
            <BasicButton
              justifyContent="center"
              margin={"0 0 0 auto"}
              width="150px"
              text={"Remove"}
              onClick={() =>
                openDimmedPopup({
                  title: "Delete Aggregator",
                  confirmText: "Delete",
                  cancelText: "Cancel",
                  size: "small",
                  buttonTwo: true,
                  form: false,
                  onConfirm: () => {
                    handleDelete(aggregator.aggregatorHash);
                    closeDimmedPopup();
                  },
                  onCancel: closeDimmedPopup,
                })
              }
            />
            <TableRow>
              <TableLabel>Hash</TableLabel>
              <TableData>{aggregator.aggregatorHash}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>Name</TableLabel>
              <TableData>{aggregator.name}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>Address</TableLabel>
              <TableData>{aggregator.address}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>Active</TableLabel>
              <TableData>{aggregator.active ? "Active" : "Inactive"}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>Heartbeat</TableLabel>
              <TableData>{aggregator.heartbeat}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>Threshold</TableLabel>
              <TableData>{aggregator.threshold}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>Absolute Threshold</TableLabel>
              <TableData>{aggregator.absoluteThreshold}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>Adapter Hash</TableLabel>
              <TableData>{aggregator.adapterId}</TableData>
            </TableRow>
            <TableRow>
              <TableLabel>Chain</TableLabel>
              <TableData>{aggregator.chainId}</TableData>
            </TableRow>
          </TableBase>
        ))
      )}
      {}
    </Container>
  );
};

export default Aggregator;
