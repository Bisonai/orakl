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
  jsonForm,
  size,
  placeholder,
}) => {
  const [isOpen, setIsOpen] = useState(true);
  const [inputJsonValue, setInputJsonValue] = useState<Record<string, string>>(
    {}
  );

  const handleConfirm = () => {
    setIsOpen(false);
    onConfirm(inputJsonValue);
  };
  const handleCancel = () => {
    setIsOpen(false);
    onCancel();
  };

  const handleJsonInputChange = (key: string) => (e: any) => {
    setInputJsonValue({ ...inputJsonValue, [key]: e.target.value });
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
            {jsonForm && (
              <PopupForm>
                {Object.keys(jsonForm).map((key) => (
                  <FormInputBase
                    autoFocus
                    type="text"
                    key={key}
                    value={inputJsonValue[key] || ""}
                    onChange={handleJsonInputChange(key)}
                    placeholder={key}
                  />
                ))}
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
