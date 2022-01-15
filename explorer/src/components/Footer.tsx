import { Box, IconButton, Link, Typography } from "@mui/material";
import { Link as RouterLink } from "gatsby";
import React from "react";
import Discord from "../images/Discord.svg";
import shape from "../images/footer/shape.svg";
import Github from "../images/Github.svg";
import logo from "../images/logo.svg";
import Medium from "../images/Medium.svg";
import Telegram from "../images/Telegram.svg";
import Twitter from "../images/Twitter.svg";
import {
  apps,
  blog,
  buidl,
  discord,
  docs,
  explorer,
  github,
  home,
  jobs,
  network,
  portal,
  telegram,
  twitter,
} from "../utils/urls";

const linkStyle = { display: "block", mr: 7.5, mb: 1.5, fontSize: 14 };
const socialIcon = {
  "& img": {
    height: 24,
    width: 24,
  },
};

const Footer = () => (
  <Box
    sx={{
      position: "relative",
      maxWidth: 1100,
      mx: "auto",
      mt: 21.5,
      mb: 20,
      borderTop: "1px solid white",
      pt: 7,
    }}
  >
    <Box
      sx={{
        position: "absolute",
        zIndex: -1,
        bottom: -20 * 8,
        transform: "translate(-50%, 0%)",
        background: `url(${shape})`,
        backgroundRepeat: "no-repeat",
        backgroundPosition: "right top -458px",
        // backgroundSize: "cover",
        width: "100%",
        height: 556,
      }}
    />
    <Box sx={{ display: "flex", flexWrap: "wrap" }}>
      <Box sx={{ pl: 2, pb: 2 }}>
        <RouterLink to={home}>
          <img src={logo} alt="Wormhole" />
        </RouterLink>
      </Box>
      <Box sx={{ flexGrow: 1 }} />
      <Box sx={{ pl: 2 }}>
        <Typography sx={{ mb: 3 }}>Navigate</Typography>
        <Box sx={{ display: "flex" }}>
          <Box>
            <Link
              component={RouterLink}
              to={apps}
              color="inherit"
              underline="hover"
              sx={linkStyle}
            >
              Apps
            </Link>
            <Link
              href={portal}
              color="inherit"
              underline="hover"
              sx={linkStyle}
            >
              Portal
            </Link>
            <Link
              component={RouterLink}
              to={buidl}
              color="inherit"
              underline="hover"
              sx={linkStyle}
            >
              Buidl
            </Link>
            <Link href={blog} color="inherit" underline="hover" sx={linkStyle}>
              Blog
            </Link>
          </Box>
          <Box>
            <Link
              href={network}
              color="inherit"
              underline="hover"
              sx={linkStyle}
            >
              Network
            </Link>
            <Link
              href={explorer}
              color="inherit"
              underline="hover"
              sx={linkStyle}
            >
              Explorer
            </Link>
            <Link href={docs} color="inherit" underline="hover" sx={linkStyle}>
              Docs
            </Link>
            <Link href={jobs} color="inherit" underline="hover" sx={linkStyle}>
              Jobs
            </Link>
          </Box>
        </Box>
      </Box>
      <Box sx={{ flexGrow: 1 }} />
      <Box sx={{ px: 2 }}>
        <Typography sx={{ mb: 3 }}>Let's be friends</Typography>
        <Box>
          <IconButton
            href={discord}
            target="_blank"
            rel="noopener noreferrer"
            sx={socialIcon}
          >
            <img src={Discord} alt="Discord" />
          </IconButton>
          <IconButton
            href={github}
            target="_blank"
            rel="noopener noreferrer"
            sx={socialIcon}
          >
            <img src={Github} alt="Github" />
          </IconButton>
          <IconButton
            href={blog}
            target="_blank"
            rel="noopener noreferrer"
            sx={socialIcon}
          >
            <img src={Medium} alt="Medium" />
          </IconButton>
          <IconButton
            href={telegram}
            target="_blank"
            rel="noopener noreferrer"
            sx={socialIcon}
          >
            <img src={Telegram} alt="Telegram" />
          </IconButton>
          <IconButton
            href={twitter}
            target="_blank"
            rel="noopener noreferrer"
            sx={socialIcon}
          >
            <img src={Twitter} alt="Twitter" />
          </IconButton>
        </Box>
      </Box>
    </Box>
  </Box>
);
export default Footer;
