import { createTheme, responsiveFontSizes } from "@material-ui/core";

export const theme = responsiveFontSizes(
  createTheme({
    palette: {
      type: "dark",
      background: {
        default: "#080808",
        paper: "#020202",
      },
      divider: "#4e4e54",
      text: {
        primary: "rgba(255,255,255,0.98)",
      },
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
    typography: {
      fontFamily: "'Sora', sans-serif",
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
