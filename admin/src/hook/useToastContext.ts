import { ToastContext } from "@/components/Common/ToastProvider";
import { useContext } from "react";

export const useToastContext = () => useContext(ToastContext);
