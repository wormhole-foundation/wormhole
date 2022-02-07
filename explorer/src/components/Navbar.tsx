import { AppBar, Hidden, Button,  Box, Link, Toolbar } from "@mui/material";
import ArrowForward from "@mui/icons-material/ArrowForward";
import { Link as RouterLink } from "gatsby";
import React from "react";
import hamburger from "../images/hamburger.svg";
import { apps, blog, buidl, portal } from "../utils/urls";
import LogoLink from "./LogoLink";

const linkStyle = { ml: 3, textUnderlineOffset: 6 };
const linkActiveStyle = { textDecoration: "underline" };

const NavBar = () => (
  <>
    {/* <Box sx={{
      display: 'flex',
      justifyContent: 'center',
      flexDirection: {xs: 'column', md:'row'},
      alignItems: 'center',
      background: '#17153f',
      textAlign: {xs: 'center', md:'left'},
      p: 2
    }}>
      <Box sx={{m:{xs:'0 0 10px 0', md:'0 40px 0 0'} }}>A $10,000,000 reward is offered for any information leading to the arrest and conviction of those responsible for the hack of Wormhole on February 2, 2022, or the recovery of the stolen assets. </Box>
      <Button
          component={RouterLink}
          to="#"
          sx={{flex: '0 0 auto'}}
          variant="outlined"
          color="inherit"
          endIcon={<ArrowForward />}
        >
            Learn More
      </Button>
    </Box> */}

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
