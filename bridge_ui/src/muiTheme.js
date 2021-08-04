import { createTheme, responsiveFontSizes } from "@material-ui/core";

export const theme = responsiveFontSizes(
  createTheme({
    palette: {
      type: "dark",
      background: {
        default: "#010114",
        paper: "#010114",
      },
      divider: "#4e4e54",
      primary: {
        main: "#0074FF",
      },
    },
    overrides: {
      MuiButton: {
        root: {
          borderRadius: 0,
        },
      },
    },
  })
);
