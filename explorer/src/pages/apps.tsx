import { Box } from "@mui/material";
import * as React from "react";
import { PageProps } from 'gatsby'
import GridWithCards from "../components/GridWithCards";
import HeroText from "../components/HeroText";
import Layout from "../components/Layout";
import { SEO } from "../components/SEO";
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
import shape from "../images/shape.png";

const AppsPage = ({ location }: PageProps) => {
  return (
    <Layout>
      <SEO
        title="Apps"
        description="Explore apps that give you the power of cross-chain movement by building on top of a core generic message-passing layer."
        pathname={location.pathname}
      />
      <Box sx={{ position: "relative", marginTop: 17 }}>
        <Box
          sx={{
            position: "absolute",
            zIndex: -2,
            bottom: '-250px',
            left: '20%',
            background: 'radial-gradient(closest-side at 50% 50%, #5189C8 0%, #5189C800 100%) ',
            transform: 'matrix(-0.19, 0.98, -0.98, -0.19, 0, 0)',
            width: 1609,
            height: 1264,
            pointerEvents: 'none',
            opacity: 0.7,
          }}
        />
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

        <HeroText
          heroSpans={["Every chain", "at once"]}
          subtitleText={[
            "Explore apps that give you",
            "the power of cross-chain movement",
            "by building on top of a",
            "core generic message-passing layer.",
          ]}
        />
      </Box>
      <Box sx={{ position: 'relative' }}>
        <Box
          sx={{
            position: "absolute",
            zIndex: -2,
            bottom: '-20%',
            background: 'radial-gradient(closest-side at 50% 50%, #E72850 0%, #E7285000 100%)',
            transform: 'transform: matrix(-0.77, -0.64, 0.64, -0.77, 0, 0)',
            right: '75%',
            width: 1494.89,
            height: 861.05,
            pointerEvents: 'none',
            opacity: 0.7,
          }}
        />
        <Box
          sx={{
            position: "absolute",
            zIndex: -2,
            top: '-20%',
            background: 'radial-gradient(closest-side at 50% 50%, #5189C8 0%, #5189C800 100%) ',
            transform: 'matrix(-0.67, 0.74, -0.74, -0.67, 0, 0)',
            left: '70%',
            width: 1709,
            height: 1690,
            pointerEvents: 'none',
            opacity: 0.64,
          }}
        />
        <Box
          sx={{
            position: "absolute",
            zIndex: -1,
            background: `url(${shape})`,
            backgroundSize: 'contain',
            top: -100,
            right: '85vw',
            width: 1012,
            height: 839,
            pointerEvents: 'none',
            display: { xs: 'none', md: 'block' },
          }}
        />
        <Box
          sx={{
            position: "absolute",
            zIndex: -1,
            background: `url(${shape})`,
            backgroundSize: 'contain',
            bottom: '-20%',
            left: "80%",
            transform: 'scaleX(-1)',
            width: 1227,
            height: 1018,
            pointerEvents: 'none',
            display: { xs: 'none', md: 'block' },
          }}
        />
        <Box sx={{ maxWidth: 1220, mx: "auto", mt: 36, px: 3.75 }}>
          <GridWithCards
            spacing={3}
            cardPaddingTop={3}
            imgOffsetRightMd="0px"
            imgOffsetTopXs="0px"
            imgOffsetTopMd="-36px"
            imgOffsetTopMdHover="-52px"
            imgPaddingBottomXs={3}
            data={[
              {
                src: lido,
                header: "Lido",
                href: "https://lido.fi/",
                size: 114,
                description:
                  "Stake in multiple networks while using the staked token for lending and yield farming.",
              },
              {
                src: pyth,
                header: "Pyth",
                size: 114,
                href: "https://pyth.network/markets/",
                description:
                  "Make smart contracts more accurate by connecting high-fidelity market data.",
              },
              {
                src: atlas,
                header: "Atlas Dex",
                href: "https://atlasdex.finance/",
                size: 114,
                description:
                  "Make faster transactions across chains to get the best exchange price.",
              },
              {
                src: mercurial,
                header: "Mercurial",
                href: "https://mercurial.finance/",
                size: 114,
                description:
                  "Make faster transactions with greater cross-chain liquidity in stable assets.",
              },
              {
                src: swim,
                header: "Swim Protocol",
                href: "https://swim.io/",
                size: 114,
                description:
                  "Swap chain-native assets without the need for wrapped assets or centralized exchanges.",
              },
              {
                src: orion,
                header: "Orion Money",
                href: "https://www.orion.money/",
                size: 114,
                description:
                  "Earn stablecoin yields on multiple chains from one centralized location.",
              },
              {
                src: tiexo,
                header: "Tiexo",
                href: "https://tiexo.com/",
                size: 114,
                description:
                  "Buy NFTs across chains from a wallet in multiple currencies.",
              },
              {
                src: bridgesplit,
                header: "Bridgesplit",
                href: "https://bridgesplit.com/",
                size: 114,
                description: "Sell, buy, or lend portions of NFTs across chains.",
              },
              {
                src: faraway,
                header: "Faraway Games",
                href: "https://faraway.gg/",
                size: 114,
                description:
                  "Validates membership to some game communities using ETH NFTs.",
              },
            ]}
          />
        </Box>
      </Box>
    </Layout>
  );
};

export default AppsPage;
