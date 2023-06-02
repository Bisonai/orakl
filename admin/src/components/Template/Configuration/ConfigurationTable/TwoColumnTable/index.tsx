import { StringLiteral } from "typescript";
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
import { useDimmedPopupContext } from "@/hook/useDimmedPopupContext";
import { useEffect, useState } from "react";

const TwoColumnTable = ({
  title,
  data,
  buttonProps,
  addTitle,
  deleteTitle,
  addConfirmText,
  deleteConfrimText,
}: {
  title: string;
  data: any;
  buttonProps: any;
  addTitle: string;
  deleteTitle: string;
  addConfirmText: string;
  deleteConfrimText: string;
}) => {
  const { openDimmedPopup, closeDimmedPopup } = useDimmedPopupContext();
  const [localData, setLocalData] = useState<any[]>(data);
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
          setLocalData((prevData) => [
            ...prevData,
            { id: prevData.length + 1, name: inputValue },
          ]);
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
      confirmText: deleteConfrimText,
      cancelText: "Cancel",
      size: "small",
      buttonTwo: true,
      onConfirm: () => {
        setLocalData((prevData) =>
          prevData.filter((item, itemIndex) => itemIndex !== index)
        );
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
          {localData.map((item: any, index: number) => (
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
          ))}
        </TableContainer>
      </TwoColumnTableContainer>
    </>
  );
};

export default TwoColumnTable;
