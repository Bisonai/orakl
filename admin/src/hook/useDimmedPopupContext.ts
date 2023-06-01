import { DimmedPopupContext } from "@/components/Common/DimmedPopupProvider";
import { useContext } from "react";

export const useDimmedPopupContext = () => useContext(DimmedPopupContext);
