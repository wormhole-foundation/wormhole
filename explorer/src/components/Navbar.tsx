import { AppBar, Hidden, Button,  Box, Link, Toolbar } from "@mui/material";
import ArrowForward from "@mui/icons-material/ArrowForward";
import { Link as RouterLink } from "gatsby";
import React from "react";
import { OutboundLink } from "gatsby-plugin-google-gtag";
import hamburger from "../images/hamburger.svg";
import { apps, blog, buidl, portal } from "../utils/urls";
import LogoLink from "./LogoLink";

const linkStyle = { ml: 3, textUnderlineOffset: 6 };
const linkActiveStyle = { textDecoration: "underline" };

const NavBar = () => (
  <>
    <Box sx={{
      display: 'flex',
      flexWrap: 'wrap',
      justifyContent: 'center',
      alignItems: 'center',
      background: '#17153f',
      textAlign: 'center',
      p: 1
    }}>
      <Box sx={{ m: '5px 10px' }}>ImmuneFi bug bounty</Box>
      <Button
          component={OutboundLink}
          href="https://www.immunefi.com/bounty/wormhole/"
          target='_blank'
          sx={{m: '5px 10px', flex: '0 0 auto'}}
          variant="outlined"
          color="inherit"
          endIcon={<ArrowForward />}
        >
            Learn More
      </Button>
    </Box>

  <AppBar
    position="static"
    sx={{ backgroundColor: "transparent" }}
    elevation={0}
  >
    <Toolbar disableGutters sx={{ mt: 2, mx: 4 }}>
      <LogoLink />
      <Box sx={{ flexGrow: 1 }} />
      <Box sx={{ display: { xs: "none", md: "block" } }}>
        <Link
          component={RouterLink}
          to={apps}
          color="inherit"
          underline="hover"
          sx={linkStyle}
          activeStyle={linkActiveStyle}
        >
          Apps
        </Link>
        <Link href={portal} color="inherit" underline="hover" sx={linkStyle}>
          Portal
        </Link>
        <Link
          component={RouterLink}
          to={buidl}
          color="inherit"
          underline="hover"
          sx={linkStyle}
          activeStyle={linkActiveStyle}
        >
          Buidl
        </Link>
        <Link href={blog} color="inherit" underline="hover" sx={linkStyle}>
          Blog
        </Link>
      </Box>
      {/* <Box sx={{ display: "flex", ml: 8 }}>
        <img src={hamburger} alt="menu" />
      </Box> */}
    </Toolbar>
  </AppBar>
  
  </>
);
export default NavBar;
