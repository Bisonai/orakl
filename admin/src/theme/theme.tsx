import { Button } from "@mui/material";
import { createTheme } from "@mui/material/styles";
import { withStyles } from "@mui/styles";

export const theme = createTheme({
  palette: {
    primary: {
      main: "#393E46",
    },
    secondary: {
      main: "#00ADB5",
    },
  },
});

export const StyledButton = withStyles({
  root: {
    fontSize: "12px",
    color: "white",
    padding: "10px",
    borderRadius: "5px",
    minWidth: "50px",
    height: "30px",
  },
})(Button);
