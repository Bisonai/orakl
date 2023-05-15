import { styled } from "@mui/material/styles";
import { theme } from "@/theme/theme";

export const AccordionContainer = styled("div")({
  width: "20%",
  height: "100%",
  marginTop: "100px",
  background: theme.palette.primary.main,
});
export const AccordionDetails = styled("div")(({ theme }) => ({
  borderTop: "1px solid rgba(0, 0, 0, .125)",
  display: "flex",
  flexDirection: "column",
}));
