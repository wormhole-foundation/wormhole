import {
  createTheme,
  CssBaseline,
  responsiveFontSizes,
  ThemeProvider,
} from "@mui/material";
import React from "react";

let theme = createTheme({
  palette: {
    mode: "dark",
    background: {
      default: "#17153f",
    },
  },
  typography: {
    fontFamily: ["Poppins", "Arial"].join(","),
    fontSize: 13,
    h1: {
      fontWeight: "bold",
    },
  },
  components: {
    MuiButton: {
      styleOverrides: {
        root: {
          borderRadius: 22,
          fontSize: 12,
          fontWeight: 700,
          letterSpacing: 1.5,
          padding: "8px 22.5px 6px",
          "&:hover .MuiButton-endIcon": {
            marginLeft: 16,
          },
        },
        contained: {
          boxShadow: "none",
          "&:hover": {
            boxShadow: "none",
          },
          "&:active": {
            boxShadow: "none",
          },
        },
        endIcon: {
          marginLeft: 12,
          transition: "margin-left 300ms",
        },
      },
    },
  },
});
theme = responsiveFontSizes(theme);

const TopLayout = ({ children }) => (
  <ThemeProvider theme={theme}>
    <CssBaseline />
    {children}
  </ThemeProvider>
);

export default TopLayout;
