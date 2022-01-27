import ArrowForward from "@mui/icons-material/ArrowForward";
import { AppBar, Box, Button, Toolbar, Typography } from "@mui/material";
import { Link as RouterLink } from "gatsby";
import * as React from "react";
import HeroText from "../components/HeroText";
import LogoLink from "../components/LogoLink";
import shape1 from "../images/index/shape1.svg";
import { home } from "../utils/urls";

// TODO: switch out shape for the 404 one

const IndexPage = () => {
  return (
    <>
      <Box
        sx={{
          position: "absolute",
          zIndex: -1,
          // transform: "translate(0px, -25%)",
          background: `url(${shape1})`,
          backgroundRepeat: "no-repeat",
          backgroundPosition: "center center",
          backgroundSize: "2070px 1155px",
          width: "100%",
          height: "100vh",
        }}
      />
      <AppBar
        position="static"
        sx={{ backgroundColor: "transparent" }}
        elevation={0}
      >
        <Toolbar disableGutters sx={{ mt: 2, mx: 4 }}>
          <LogoLink />
          <Box sx={{ flexGrow: 1 }} />
          <Box sx={{ display: { xs: "none", md: "block" } }}>
            <Button
              component={RouterLink}
              to={home}
              endIcon={<ArrowForward />}
              color="inherit"
              variant="outlined"
            >
              Back to Home
            </Button>
          </Box>
        </Toolbar>
      </AppBar>
      <Box
        sx={{
          position: "fixed",
          top: "50%",
          left: "50%",
          width: "100%",
          transform: "translate(-50%, -50%)",
        }}
      >
        <HeroText
          maxWidth={800}
          heroSpans={["404", "Lightyears Away"]}
          subtitleText="Looks like you found your way to a black hole rather than Wormhole. Hit the reverse thrusters."
        />
      </Box>
      <Box
        sx={{
          position: "absolute",
          bottom: 40,
          textAlign: "center",
          width: "100%",
        }}
      >
        <Typography variant="body2">
          2022 &copy; Wormhole. All Rights Reserved.
        </Typography>
      </Box>
    </>
  );
};

export default IndexPage;
