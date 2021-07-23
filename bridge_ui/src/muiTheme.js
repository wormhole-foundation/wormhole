import { createTheme, responsiveFontSizes } from "@material-ui/core";

export const theme = responsiveFontSizes(
  createTheme({
    palette: {
      type: "dark",
      background: {
        default: "#010114",
        paper: "#010114",
      },
      divider: "#FFFFFF",
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
