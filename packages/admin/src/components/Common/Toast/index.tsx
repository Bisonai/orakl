import { IToast } from "@/utils/types";
import React from "react";
import {
  CloseButtonWrap,
  TextBase,
  TextTitleBase,
  TextWrap,
  ToastWrap,
} from "./styled";

import { StyledButton } from "@/theme/theme";

const Toast = ({
  title,
  content,
  id,
  type,
  onClose,
  ...props
}: IToast & { onClose: (e: any) => any }) => {
  return (
    <>
      <ToastWrap
        key={id}
        id={id ? `${id}` : undefined}
        property={type}
        {...props}
      >
        <TextWrap>
          <TextTitleBase>{title}</TextTitleBase>
          <TextBase>{content}</TextBase>
        </TextWrap>
        <CloseButtonWrap>
          <StyledButton onClick={onClose} color="secondary" variant="contained">
            Close
          </StyledButton>
        </CloseButtonWrap>
      </ToastWrap>
    </>
  );
};

export default Toast;
