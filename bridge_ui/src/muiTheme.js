import { createTheme, responsiveFontSizes } from "@material-ui/core";

export const COLORS = {
  lightGreen: "rgba(51, 242, 223, 1)",
  green: "#00EFD8",
  blue: "#0074FF",
  blueWithTransparency: "rgba(0, 116, 255, 0.8)",
  greenWithTransparency: "rgba(0,239,216,0.8)",
  nearBlack: "#010114",
  nearBlackWithMinorTransparency: "rgba(0,0,0,.97)",
};

export const theme = responsiveFontSizes(
  createTheme({
    palette: {
      type: "dark",
      background: {
        default: COLORS.nearBlack,
        paper: COLORS.nearBlack,
      },
      divider: "#4e4e54",
      text: {
        primary: "rgba(255,255,255,0.98)",
      },
      primary: {
        main: COLORS.blueWithTransparency, // #0074FF
      },
      secondary: {
        main: COLORS.greenWithTransparency, // #00EFD8
        light: COLORS.lightGreen,
      },
      error: {
        main: "#FD3503",
      },
    },
    typography: {
      fontFamily: "'Sora', sans-serif",
      h2: {
        fontWeight: "700",
      },
      h4: {
        fontWeight: "500",
      },
    },
    overrides: {
      MuiButton: {
        root: {
          borderRadius: "5px",
          textTransform: "none",
        },
      },
    },
  })
);
