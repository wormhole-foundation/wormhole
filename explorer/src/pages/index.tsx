import ArrowForward from "@mui/icons-material/ArrowForward";
import { Box, Button, Typography } from "@mui/material";
import * as React from "react";
import AvoidBreak from "../components/AvoidBreak";
import GridWithCards from "../components/GridWithCards";
import Layout from "../components/Layout";
import apps from "../images/index/apps.png";
import blob from "../images/index/blob.svg";
import cross from "../images/index/cross.svg";
import cube from "../images/index/cube.svg";
import portal from "../images/index/portal.png";
import protocols from "../images/index/protocols.png";
import shape1 from "../images/index/shape1.svg";

const featuredNumber = { fontSize: 42, fontWeight: "bold" };

const IndexPage = () => {
  return (
    <Layout>
      <Box sx={{ position: "relative", marginTop: 21 }}>
        <Box
          sx={{
            position: "absolute",
            zIndex: -1,
            transform: "translate(0px, -25%)",
            background: `url(${shape1})`,
            backgroundRepeat: "no-repeat",
            backgroundPosition: "top -120px left -27px",
            backgroundSize: "2070px 1155px",
            width: "100%",
            height: 1155,
          }}
        />
        <Box sx={{ m: "auto", maxWidth: 600, textAlign: "center" }}>
          <Typography variant="h1">
            <AvoidBreak spans={["The best of", "blockchains"]} />
          </Typography>
          <Typography sx={{ marginTop: 2 }}>
            Move information and value anywhere.
          </Typography>
          <Box
            sx={{
              width: "calc( 100& - 16px )",
              mx: "auto",
              mt: 15.5,
              display: "flex",
              flexWrap: "wrap",
              justifyContent: "center",
            }}
          >
            <Box
              sx={{
                mt: 2,
                mx: 1,
                display: "flex",
                alignItems: "center",
                justifyContent: "space-evenly",
                flexBasis: "calc(33.33333% - 16px)",
                borderTop: "1px solid white",
              }}
            >
              <Typography sx={featuredNumber}>$1bn</Typography>
              <Typography variant="body2">in TVL</Typography>
            </Box>
            <Box
              sx={{
                mt: 2,
                mx: 1,
                display: "flex",
                alignItems: "center",
                justifyContent: "space-evenly",
                flexBasis: "calc(33.33333% - 16px)",
                borderTop: "1px solid white",
              }}
            >
              <Typography sx={featuredNumber}>6</Typography>
              <Typography variant="body2">chain integrations</Typography>
            </Box>
            <Box
              sx={{
                mt: 2,
                mx: 1,
                display: "flex",
                alignItems: "center",
                justifyContent: "space-evenly",
                flexBasis: "calc(33.33333% - 16px)",
                borderTop: "1px solid white",
              }}
            >
              <Typography sx={featuredNumber}>50k+</Typography>
              <Typography variant="body2">txs</Typography>
            </Box>
          </Box>
        </Box>
      </Box>
      <Box
        sx={{
          display: "flex",
          flexWrap: "wrap",
          maxWidth: 1220,
          px: 3.75,
          mt: 50,
          mx: "auto",
          alignItems: "center",
          justifyContent: "center",
        }}
      >
        <Box
          sx={{
            flexBasis: { xs: "100%", md: "50%" },
            flexGrow: 1,
          }}
        >
          <Box sx={{ maxWidth: 460, mx: "auto" }}>
            <Typography variant="h3">
              <Box component="span" sx={{ color: "#FFCE00" }}>
                Protocol:
              </Box>
              <Box component="span" sx={{ display: "inline-block" }}>
                the core layer
              </Box>
            </Typography>
            <Typography sx={{ mt: 2 }}>
              It's the foundation that the ecosystem of apps is built on top of.
            </Typography>
            <Button
              sx={{ mt: 3 }}
              variant="outlined"
              color="inherit"
              endIcon={<ArrowForward />}
            >
              Learn More
            </Button>
          </Box>
        </Box>
        <Box
          sx={{
            mt: { xs: 8, md: null },
            flexBasis: { xs: "100%", md: "50%" },
            textAlign: "center",
            flexGrow: 1,
          }}
        >
          <Box
            component="img"
            src={protocols}
            alt=""
            sx={{ maxWidth: "100%" }}
          />
        </Box>
      </Box>
      <Box
        sx={{
          display: "flex",
          flexWrap: "wrap-reverse",
          maxWidth: 1220,
          px: 3.75,
          mt: 15.5,
          mx: "auto",
          alignItems: "center",
          justifyContent: "center",
        }}
      >
        <Box
          sx={{
            mt: { xs: 8, md: null },
            flexBasis: { xs: "100%", md: "60%" },
            textAlign: "center",
            flexGrow: 1,
          }}
        >
          <Box component="img" src={apps} alt="" sx={{ maxWidth: "100%" }} />
        </Box>
        <Box sx={{ flexBasis: { xs: "100%", md: "40%" }, flexGrow: 1 }}>
          <Box sx={{ maxWidth: 460, mx: "auto" }}>
            <Typography variant="h3">
              <Box component="span" sx={{ color: "#FFCE00" }}>
                Apps:
              </Box>
              <Box component="span"> endless possibilities</Box>
            </Typography>
            <Typography sx={{ mt: 2 }}>
              Apps can now live across chains and integrate the best of each.
            </Typography>
            <Button
              sx={{ mt: 3 }}
              variant="outlined"
              color="inherit"
              endIcon={<ArrowForward />}
            >
              Learn More
            </Button>
          </Box>
        </Box>
      </Box>
      <Box
        sx={{
          display: "flex",
          flexWrap: "wrap",
          maxWidth: 1220,
          px: 3.75,
          mt: 15.5,
          mx: "auto",
          alignItems: "center",
          justifyContent: "center",
        }}
      >
        <Box
          sx={{
            flexBasis: { xs: "100%", md: "45%" },
            flexGrow: 1,
          }}
        >
          <Box sx={{ maxWidth: 460, mx: "auto" }}>
            <Typography variant="h3">
              <Box component="span" sx={{ color: "#FFCE00" }}>
                Portal:
              </Box>
              <Box component="span" sx={{ display: "inline-block" }}>
                a token bridge
              </Box>
            </Typography>
            <Typography sx={{ mt: 2 }}>
              Tramsfer tokens across chains and enjoy $1bn+ of liquidity
            </Typography>
            <Button
              sx={{ mt: 3 }}
              variant="outlined"
              color="inherit"
              endIcon={<ArrowForward />}
            >
              Learn More
            </Button>
          </Box>
        </Box>
        <Box
          sx={{
            mt: { xs: 8, md: null },
            flexBasis: { xs: "100%", md: "55%" },
            textAlign: "center",
            flexGrow: 1,
          }}
        >
          <Box component="img" src={portal} alt="" sx={{ maxWidth: "100%" }} />
        </Box>
      </Box>
      <Box sx={{ textAlign: "center", mt: 12.5 }}>
        <Typography variant="h3">
          <Box component="span" sx={{ color: "#FFCE00" }}>
            Cross-chain
          </Box>
          <Box component="span"> everything</Box>
        </Typography>
        <Typography sx={{ mt: 2, maxWidth: 480, mx: "auto" }}>
          Each blockchain has a distinct strength. Wormhole lets you get the
          best out of every blockchain without compromise.
        </Typography>
      </Box>
      <Box sx={{ maxWidth: 1220, mx: "auto", mt: 12, px: 3.75 }}>
        <GridWithCards
          data={[
            {
              src: cross,
              header: "Never stop expanding",
              description:
                "Chains, information, and users are growing everyday. Build on a protocol that is set up to scale, with no limits, right from the start.",
            },
            {
              src: blob,
              header: "Explore and experiment",
              description:
                "Now is the time to explore and experiment. The only limit to what you're able to build is your imagination.",
            },
            {
              src: cube,
              header: "Power your project",
              description:
                "Join the growing list of projects that are composing, raising, and succeeding with Wormhole core layer.",
            },
          ]}
        />
      </Box>
    </Layout>
  );
};

export default IndexPage;
