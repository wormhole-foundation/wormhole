import {
  createTheme,
  CssBaseline,
  responsiveFontSizes,
  ThemeProvider,
} from "@mui/material";
import TimeAgo from "javascript-time-ago";
import en from "javascript-time-ago/locale/en";
import React from "react";
import { NetworkContextProvider } from "../../src/contexts/NetworkContext";

TimeAgo.addDefaultLocale(en);

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
    MuiCssBaseline: {
      styleOverrides: {
        ul: {
          paddingLeft: "0px",
        },
        "*": {
          scrollbarWidth: "thin",
          scrollbarColor: `#4e4e54 rgba(0,0,0,.25)`,
        },
        "*::-webkit-scrollbar": {
          width: "8px",
          height: "8px",
          backgroundColor: "rgba(0, 0, 0, 0.25)",
        },
        "*::-webkit-scrollbar-thumb": {
          backgroundColor: "#4e4e54",
          borderRadius: "4px",
        },
        "*::-webkit-scrollbar-corner": {
          // this hides an annoying white box which appears when both scrollbars are present
          backgroundColor: "transparent",
        },
      },
    },
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
    MuiOutlinedInput: {
      styleOverrides: {
        notchedOutline: {
          borderRadius: 24,
        },
      },
    },
    MuiSelect: {
      styleOverrides: {
        select: {
          paddingTop: 8,
          paddingRight: "40px!important",
          paddingBottom: 8,
          paddingLeft: 20,
        },
      },
    },
  },
});
theme = responsiveFontSizes(theme);

const TopLayout = ({ children }) => (
  <ThemeProvider theme={theme}>
    <CssBaseline />
    <NetworkContextProvider>{children}</NetworkContextProvider>
  </ThemeProvider>
);

export default TopLayout;
