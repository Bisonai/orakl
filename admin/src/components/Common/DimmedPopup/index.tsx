import { useState, useEffect } from "react";
import {
  PopupContainer,
  PopupContent,
  PopupTitle,
  PopupButton,
  PopupForm,
  FormInputBase,
} from "./styled";
import { IDimmedPopupProps } from "@/utils/types";

const DimmedPopup: React.FC<IDimmedPopupProps> = ({
  title,
  confirmText,
  cancelText,
  onConfirm,
  onCancel,
  buttonTwo,
  form,
  size,
}) => {
  const [isOpen, setIsOpen] = useState(true);
  const [inputValue, setInputValue] = useState("");
  const handleConfirm = () => {
    setIsOpen(false);
    onConfirm(inputValue);
  };
  const handleCancel = () => {
    setIsOpen(false);
    onCancel();
  };
  const handleInputChange = (e: any) => {
    setInputValue(e.target.value);
  };
  const handleContainerClick = (e: any) => {
    if (e.target === e.currentTarget) {
      handleCancel();
    }
  };
  useEffect(() => {
    const handleKeyDown = (e: any) => {
      if (e.key === "Escape") {
        handleCancel();
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, []);
  const buttons = buttonTwo ? (
    <>
      <button className="btn-cancel" onClick={handleCancel}>
        {cancelText}
      </button>
      <button className="btn-confirm" onClick={handleConfirm}>
        {confirmText}
      </button>
    </>
  ) : (
    <button className="btn" onClick={handleConfirm}>
      {confirmText}
    </button>
  );

  return (
    <>
      {isOpen && (
        <PopupContainer onClick={handleContainerClick}>
          <PopupContent size={size}>
            <PopupTitle size={size}>{title} </PopupTitle>
            {form && (
              <PopupForm>
                <FormInputBase
                  autoFocus
                  type="text"
                  value={inputValue}
                  onChange={handleInputChange}
                />
              </PopupForm>
            )}
            <PopupButton>{buttons}</PopupButton>
          </PopupContent>
        </PopupContainer>
      )}
    </>
  );
};

export default DimmedPopup;
