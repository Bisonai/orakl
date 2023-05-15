import * as React from "react";
import { styled } from "@mui/material/styles";
import MuiAccordionDetails from "@mui/material/AccordionDetails";
import Typography from "@mui/material/Typography";
import { AccordionContainer, AccordionDetails } from "./styled";
import AccordionWrap from "./AccordionWrap";
import AccordionSummary from "./AccordionSummary";
import BasicButton from "../BasicButton";

export default function CustomizedAccordions() {
  const [expanded, setExpanded] = React.useState<string | false>("panel1");

  const handleChange =
    (panel: string) => (event: React.SyntheticEvent, newExpanded: boolean) => {
      setExpanded(newExpanded ? panel : false);
    };

  return (
    <AccordionContainer>
      <AccordionWrap
        expanded={expanded === "panel1"}
        onChange={handleChange("panel1")}
      >
        <AccordionSummary aria-controls="panel1d-content" id="panel1d-header">
          <Typography>First data</Typography>
        </AccordionSummary>
        <AccordionDetails>
          <BasicButton text={"adfa"} />
          <BasicButton text={"adfa"} />
          <BasicButton text={"adfa"} />
        </AccordionDetails>
      </AccordionWrap>
      <AccordionWrap
        expanded={expanded === "panel2"}
        onChange={handleChange("panel2")}
      >
        <AccordionSummary aria-controls="panel2d-content" id="panel2d-header">
          <Typography>Second data</Typography>
        </AccordionSummary>
        <AccordionDetails>
          <BasicButton text={"adfa"} />
          <BasicButton text={"adfa"} />
          <BasicButton text={"adfa"} />
        </AccordionDetails>
      </AccordionWrap>
      <AccordionWrap
        expanded={expanded === "panel3"}
        onChange={handleChange("panel3")}
      >
        <AccordionSummary aria-controls="panel3d-content" id="panel3d-header">
          <Typography>Third data</Typography>
        </AccordionSummary>
        <AccordionDetails>
          <BasicButton text={"adfa"} />
          <BasicButton text={"adfa"} />
          <BasicButton text={"adfa"} />
        </AccordionDetails>
      </AccordionWrap>
    </AccordionContainer>
  );
}
