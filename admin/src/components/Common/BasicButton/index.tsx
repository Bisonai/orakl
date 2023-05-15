import * as React from "react";
import Button, { ButtonProps } from "@mui/material/Button";

interface BasicButtonProps extends ButtonProps {
  text: string;
  disabled?: boolean;
  justifyContent?: string;
  width?: string;
}

export default function BasicButton({
  text,
  onClick,
  color = "primary",
  variant = "contained",
  disabled = false,
  justifyContent = "flex-start",
  width = "100%",
  ...rest
}: BasicButtonProps) {
  return (
    <Button
      onClick={onClick}
      variant={variant}
      color={color}
      style={{ width, justifyContent }}
      disabled={disabled}
      {...rest}
    >
      {text}
    </Button>
  );
}
