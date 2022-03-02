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
import bg from "../../src/images/bg.svg";
import { Helmet } from "react-helmet";

import Suisse from "../../src/fonts/SuisseBPIntlBold.woff2";

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
    body1: {
      fontWeight: 300,
    },
    body2: {
      fontWeight: 300,
    },
    h1: {
      fontWeight: "bold",
      fontFamily: ["Suisse BP Intl", "Arial"],
      lineHeight: 0.9,
      letterSpacing: -2.7,
    },
    h3: {
      fontSize: 49,
      fontWeight: "bold",
      fontFamily: "Suisse BP Intl",
      lineHeight: 0.9,
      letterSpacing: -1.47,
    },
    h4: {
      fontSize: 40,
      fontWeight: "bold",
      fontFamily: "Suisse BP Intl",
      letterSpacing: -1.2,
      lineHeight: 0.9,
    },
    caption: {
      textTransform: "uppercase",
      fontSize: 8,
      letterSpacing: 2,
      fontFamily: "Suisse BP Intl",
      fontWeight: 400,
      display: "block",
      marginTop: 10,
    },
  },

  components: {
    MuiCssBaseline: {
      styleOverrides: {
        body: {
          overscrollBehaviorY: "none",
          backgroundColor: "#17153f",
          backgroundImage: `url(${bg})`,
          backgroundPosition: "top center",
          backgroundRepeat: "repeat-y",
          backgroundSize: "120%",
        },
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
          padding: "8px 22.5px 8px",
          "&:hover .MuiButton-endIcon": {
            transform: "translateX(4px)",
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
          transition: "transform 300ms",
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
    <Helmet>
      <link
        rel="preload"
        as="font"
        href={Suisse}
        type="font/woff2"
        crossOrigin="anonymous"
      />
    </Helmet>
    <CssBaseline />
    <NetworkContextProvider>{children}</NetworkContextProvider>
  </ThemeProvider>
);

export default TopLayout;
