import { createContext, useState } from "react";
import { IDimmedPopupContext, IDimmedPopupProps } from "@/utils/types";
import DimmedPopup from "../DimmedPopup";

interface IDimmedPopupProviderProps {
  children: React.ReactNode;
}

export const DimmedPopupContext = createContext<IDimmedPopupContext>({
  isOpen: false,
  openDimmedPopup: () => {},
  closeDimmedPopup: () => {},
  inputJsonValue: {},
  setInputJsonValue: () => {},
});

export default function DimmedPopupProvider({
  children,
}: IDimmedPopupProviderProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [DimmedPopupProps, setDimmedPopupProps] =
    useState<IDimmedPopupProps | null>(null);
  const [inputJsonValue, setInputJsonValue] = useState({});

  const openDimmedPopup = (props: IDimmedPopupProps) => {
    setDimmedPopupProps(props);
    setIsOpen(true);
  };

  const closeDimmedPopup = () => {
    setIsOpen(false);
  };

  const contextValue: IDimmedPopupContext = {
    isOpen,
    openDimmedPopup,
    closeDimmedPopup,
    inputJsonValue,
    setInputJsonValue,
  };

  return (
    <DimmedPopupContext.Provider value={contextValue}>
      {children}
      {isOpen && DimmedPopupProps && <DimmedPopup {...DimmedPopupProps} />}
    </DimmedPopupContext.Provider>
  );
}
