import { createTheme, responsiveFontSizes } from "@material-ui/core";

export const theme = responsiveFontSizes(
  createTheme({
    palette: {
      type: "dark",
      background: {
        default: "#010114",
        paper: "rgba(0, 0, 0, 0.2)",
      },
      divider: "#4e4e54",
      primary: {
        main: "rgba(0, 116, 255, 0.8)", // #0074FF
      },
      secondary: {
        main: "rgb(0,239,216,0.8)", // #00EFD8
        light: "rgb(51, 242, 223, 1)",
      },
      error: {
        main: "#FD3503",
      },
    },
    overrides: {
      MuiButton: {
        root: {
          borderRadius: 0,
          textTransform: "none",
        },
      },
    },
  })
);
