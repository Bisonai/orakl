import * as React from "react";
import { styled } from "@mui/material/styles";
import MuiAccordion, { AccordionProps } from "@mui/material/Accordion";

const AccordionWrap = styled((props: AccordionProps) => (
  <MuiAccordion disableGutters elevation={0} square {...props} />
))(({ theme }) => ({
  color: "#fef1cc",
  border: `1px solid ${theme.palette.divider}`,
  backgroundColor: "rgba(60, 67, 74, 1)",
  "&:not(:last-child)": {
    borderBottom: 0,
  },
  "&:before": {
    display: "none",
  },
}));

export default AccordionWrap;
