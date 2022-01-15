import { Box, Typography } from "@mui/material";
import * as React from "react";
import AvoidBreak from "../components/AvoidBreak";
import GridWithCards from "../components/GridWithCards";
import Layout from "../components/Layout";
import atlas from "../images/apps/atlas.png";
import bridgesplit from "../images/apps/bridgesplit.png";
import faraway from "../images/apps/faraway.png";
import lido from "../images/apps/lido.png";
import mercurial from "../images/apps/mercurial.png";
import orion from "../images/apps/orion.png";
import pyth from "../images/apps/pyth.png";
import swim from "../images/apps/swim.png";
import tiexo from "../images/apps/tiexo.png";
import shape1 from "../images/index/shape1.svg";

const AppsPage = () => {
  return (
    <Layout>
      <Box sx={{ position: "relative", marginTop: 17 }}>
        <Box
          sx={{
            position: "absolute",
            zIndex: -1,
            transform: "translate(0px, -25%) scaleX(-1)",
            background: `url(${shape1})`,
            backgroundRepeat: "no-repeat",
            backgroundPosition: "top -540px center",
            backgroundSize: "2070px 1155px",
            width: "100%",
            height: 1155,
          }}
        />
        <Box sx={{ m: "auto", maxWidth: 1155, textAlign: "center" }}>
          <Typography variant="h1">
            <AvoidBreak spans={["Every chain", "at once"]} />
          </Typography>
          <Typography sx={{ marginTop: 3.25 }}>
            Explore apps that give you the power of cross-chain movement.
          </Typography>
        </Box>
      </Box>
      <Box sx={{ maxWidth: 1220, mx: "auto", mt: 36, px: 3.75 }}>
        <GridWithCards
          spacing={3}
          cardPaddingTop={3}
          imgOffsetRightMd="0px"
          imgOffsetTopXs="0px"
          imgOffsetTopMd="-36px"
          imgPaddingBottomXs={3}
          data={[
            {
              src: lido,
              header: "Lido",
              description:
                "Stake in multiple networks while using the staked token for lending and yield farming.",
            },
            {
              src: pyth,
              header: "Pyth",
              description:
                "Make smart contracts more accurate by connecting high-fidelity",
            },
            {
              src: atlas,
              header: "Atlas Dex",
              description:
                "Make faster transactions across chains to get the best exchange price.",
            },
            {
              src: mercurial,
              header: "Mercurial",
              description:
                "Make faster transactions with greater cross-chain liquidity in stable assets on Solana.",
            },
            {
              src: swim,
              header: "Swim Protocol",
              description:
                "Swap chain-native assets easily without the need for wrapped assets or centralized exchanges.",
            },
            {
              src: orion,
              header: "Orion Money",
              description:
                "Earn stablecoin yields on multiple chains from one centralized location.",
            },
            {
              src: tiexo,
              header: "Tiexo",
              description:
                "Buy NFTs across chains from a wallet in multiple currencies.",
            },
            {
              src: bridgesplit,
              header: "Bridgesplit",
              description: "Sell, buy, or lend portions of NFTs across chains.",
            },
            {
              src: faraway,
              header: "Faraway Games",
              description:
                "Validates membership to some game communities using ETH NFTs.",
            },
          ]}
        />
      </Box>
    </Layout>
  );
};

export default AppsPage;
