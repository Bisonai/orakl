"use client";

import { IToast, IToastContextProps, ToastType } from "@/utils/types";
import React, { useState } from "react";
import styled from "styled-components";
import Toast from "../Toast";

const ToastContext: React.Context<IToastContextProps> =
  React.createContext<IToastContextProps>({
    toasts: [],
    addToast: function (toast: IToast) {
      return;
    },
    updateToast: function (toast: IToast) {
      return;
    },
    removeToast: function (id: string | number) {
      return;
    },
    clearToast: function () {
      return;
    },
  });

export const ToastsTemplate = styled.div`
  position: fixed;
  top: 0;
  right: 0;
  z-index: 1000;
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  padding: 0 16px;
  pointer-events: none;
`;

export const ToastsWrap = styled.div`
  position: relative;
  display: flex;
  flex-direction: column-reverse;
  align-items: flex-end;
  justify-content: flex-start;
  width: 100%;
  height: 100%;
  padding: 16px 0;
  row-gap: 10px;
`;

const MemoizedToastsTemplate = React.memo(ToastsTemplate);

const ToastContextProvider = ({
  children,
}: {
  children: React.ReactNode;
}): JSX.Element => {
  const ref = React.useRef<HTMLDivElement>(null);
  const [toasts, setToasts] = useState<(IToast & { id: number | string })[]>(
    []
  );

  const addToast = (toast: IToast) => {
    const newToast = { ...toast, id: toast.id ? toast.id : toasts.length + 1 };
    setToasts((toasts) => [...toasts, newToast]);
    if (ref.current) {
      setTimeout(() => {
        removeToast(newToast.id);
      }, 3000);
    }
  };

  const updateToast = (toast: IToast) => {
    setToasts((prevToasts) =>
      prevToasts.map((t) => {
        if (t.id === toast.id) {
          return { ...t, ...toast };
        }
        return t;
      })
    );
  };

  const removeToast = (id: string | number) => {
    if (ref.current) {
      const toastElement = ref.current.querySelector(`#toast-${id}`);
      if (toastElement) {
        toastElement.classList.add("fadeOut");
      }
    }
    setTimeout(() => {
      setToasts((prevToasts) => prevToasts.filter((t) => t.id !== id));
    }, 1000);
  };

  const clearToast = () => {
    toasts.map((toast) => removeToast(toast.id));
  };
  return (
    <ToastContext.Provider
      value={{ toasts, addToast, updateToast, removeToast, clearToast }}
    >
      {children}
      <MemoizedToastsTemplate ref={ref}>
        <ToastsWrap>
          {toasts.map((toast, index) => {
            return (
              <Toast
                key={`toast-${toast.id}-${index}`}
                id={`toast-${toast.id}`}
                title={toast.title}
                content={toast.content}
                type={toast.type || ToastType.SUCCESS}
                onClose={() => removeToast(toast.id)}
              />
            );
          })}
        </ToastsWrap>
      </MemoizedToastsTemplate>
    </ToastContext.Provider>
  );
};

export default ToastContextProvider;

export { ToastContext };
