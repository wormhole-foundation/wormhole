import { AppBar, Box, Link, Toolbar } from "@mui/material";
import { Link as RouterLink } from "gatsby";
import React from "react";
import logo from "../images/logo.svg";
import hamburger from "../images/hamburger.svg";

const linkStyle = { ml: 3 };

const NavBar = () => (
  <AppBar
    position="static"
    sx={{ backgroundColor: "transparent" }}
    elevation={0}
  >
    <Toolbar disableGutters sx={{ mt: 2, mx: 4 }}>
      <RouterLink to="/" style={{ display: "flex" }}>
        <img src={logo} alt="Wormhole" />
      </RouterLink>
      <Box sx={{ flexGrow: 1 }} />
      <Box sx={{ display: { xs: "none", md: "block" } }}>
        <Link
          component={RouterLink}
          to="/apps"
          color="inherit"
          underline="hover"
          sx={linkStyle}
        >
          Apps
        </Link>
        <Link
          href="https://wormholebridge.com/"
          color="inherit"
          underline="hover"
          sx={linkStyle}
        >
          Portal
        </Link>
        <Link
          component={RouterLink}
          to="/buidl"
          color="inherit"
          underline="hover"
          sx={linkStyle}
        >
          Buidl
        </Link>
        <Link
          href="https://wormholecrypto.medium.com/"
          color="inherit"
          underline="hover"
          sx={linkStyle}
        >
          Blog
        </Link>
      </Box>
      <Box sx={{ display: "flex", ml: 8 }}>
        <img src={hamburger} alt="menu" />
      </Box>
    </Toolbar>
  </AppBar>
);
export default NavBar;
