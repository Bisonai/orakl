import {
  TwoColumnTableContainer,
  IDTitleBase,
  NameTitleBase,
  TableBase,
  IDDataBase,
  NameDataBase,
  ConfigTitleBase,
  HeaderBase,
  TableContainer,
  TwoColumnTableHeaderBase,
} from "./styled";
import BasicButton from "@/components/Common/BasicButton";
import { IsLoadingBase } from "@/components/Template/BullMonitor/DetailTable/styled";
import { useDimmedPopupContext } from "@/hook/useDimmedPopupContext";
import { useEffect, useState } from "react";

const TwoColumnTable = ({
  title,
  data,
  buttonProps,
  addTitle,
  deleteTitle,
  addConfirmText,
  deleteConfirmText,
  onAdd,
  onDelete,
}: {
  title: string;
  data: any;
  buttonProps: any;
  addTitle: string;
  deleteTitle: string;
  addConfirmText: string;
  deleteConfirmText: string;
  onAdd?: (newData: any) => void;
  onDelete?: (id: string | number) => void;
}) => {
  const { openDimmedPopup, closeDimmedPopup } = useDimmedPopupContext();
  const [localData, setLocalData] = useState<any[]>(data || []);

  useEffect(() => {
    setLocalData(data);
  }, [data]);

  const handleAddBtn = () => {
    openDimmedPopup({
      title: addTitle,
      confirmText: addConfirmText,
      cancelText: "Cancel",
      size: "medium",
      buttonTwo: true,
      onConfirm: (inputValue?: string) => {
        if (inputValue) {
          const newData = { id: localData.length + 1, name: inputValue };

          setLocalData((prevData) => [...prevData, newData]);
          onAdd && onAdd(newData.name);
        }
        closeDimmedPopup();
      },
      onCancel: closeDimmedPopup,
      form: true,
    });
  };
  const handleDeleteBtn = (index: number) => {
    openDimmedPopup({
      title: deleteTitle,
      confirmText: deleteConfirmText,
      cancelText: "Cancel",
      size: "small",
      buttonTwo: true,
      onConfirm: () => {
        const deletedItem = localData[index];
        setLocalData((prevData) =>
          prevData.filter((item, itemIndex) => itemIndex !== index)
        );
        onDelete && onDelete(deletedItem.id);
        closeDimmedPopup();
      },
      onCancel: closeDimmedPopup,
      form: false,
    });
  };

  return (
    <>
      <TwoColumnTableContainer>
        <TwoColumnTableHeaderBase>
          <ConfigTitleBase>{title}</ConfigTitleBase>
          <BasicButton {...buttonProps} onClick={handleAddBtn} />
        </TwoColumnTableHeaderBase>
        <TableContainer>
          <HeaderBase>
            <IDTitleBase>ID</IDTitleBase>
            <NameTitleBase>Name</NameTitleBase>
          </HeaderBase>
          {localData?.length === 0 ? (
            <IsLoadingBase>Loading... Please wait a moment</IsLoadingBase>
          ) : (
            localData &&
            localData.map((item: any, index: number) => (
              <TableBase key={index}>
                <IDDataBase>{item.id}</IDDataBase>
                <NameDataBase>{item.name}</NameDataBase>
                <BasicButton
                  text={"Remove"}
                  width="80px"
                  justifyContent="center"
                  height="40px"
                  margin="0 30px 0 auto"
                  onClick={() => handleDeleteBtn(index)}
                  background="gray"
                />
              </TableBase>
            ))
          )}
        </TableContainer>
      </TwoColumnTableContainer>
    </>
  );
};

export default TwoColumnTable;
