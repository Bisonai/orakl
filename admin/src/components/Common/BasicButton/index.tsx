import * as React from "react";
import Button, { ButtonProps } from "@mui/material/Button";

interface BasicButtonProps extends ButtonProps {
  text: string;
  disabled?: boolean;
  justifyContent?: string;
  width?: string;
  height?: string;
  margin?: string | number;
  selected?: boolean;
  background?: string;
}

export default function BasicButton({
  text,
  onClick,
  color = "primary",
  variant = "contained",
  disabled = false,
  justifyContent = "flex-start",
  width = "100%",
  height,
  margin,
  background,
  selected = false,
  ...rest
}: BasicButtonProps) {
  return (
    <Button
      onClick={onClick}
      variant={variant}
      color={selected ? "secondary" : "primary"}
      style={{ width, justifyContent, margin, background, height, color }}
      disabled={disabled}
      {...rest}
    >
      {text}
    </Button>
  );
}
