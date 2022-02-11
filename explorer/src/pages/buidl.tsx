import {
  Timeline,
  TimelineConnector,
  TimelineContent,
  TimelineItem,
  TimelineSeparator,
} from "@mui/lab";
import { Box, Typography } from "@mui/material";
import * as React from "react";
import { PageProps } from 'gatsby'
import AvoidBreak from "../components/AvoidBreak";
import HeroText from "../components/HeroText";
import Layout from "../components/Layout";
import network from "../images/buidl/network.svg";
import shape from "../images/buidl/shape.svg";
import stack from "../images/buidl/stack.svg";
import shape1 from "../images/index/shape1.svg";
import { SEO } from "../components/SEO";
import shapes from "../images/shape.png";

const BuidlPage = ({ location }: PageProps) => {
  return (
    <Layout>
      <SEO
        title="BUIDL"
        description="One integration to rule them all. Access every chain at once with our SDK."
        pathname={location.pathname}
      />
      <Box sx={{ position: "relative", marginTop: 17 }}>
        <Box
            sx={{
              position: "absolute",
              zIndex: -2,
              bottom: '-220px',
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
            transform: "translate(0px, -25%)",
            background: `url(${shape1})`,
            backgroundRepeat: "no-repeat",
            backgroundPosition: "top -500px center",
            backgroundSize: "2070px 1155px",
            width: "100%",
            height: 1155,
          }}
        />
        <HeroText
          heroSpans={["One integration", "to rule them all"]}
          subtitleText="Access every chain at once with our SDK."
        />
      </Box>
      <Box sx={{ textAlign: "center", mt: 25.75, px: 2 }}>
        <Typography variant="h3">
          <Box component="span" sx={{ color: "#FFCE00" }}>
            Cross-chain{" "}
          </Box>
          <Box component="span">everything</Box>
        </Typography>
        <Typography sx={{ mt: 2, maxWidth: 860, mx: "auto" }}>
          Atomic Swaps | Governance | IDO Launchpad | NFT Marketplace |
          Aggregation | Rebasing Lending, Borrowing, &amp; Saving | Oracle Data
          | Liquidity | Staking Tokens | Fractional Ownership NFT Collateral |
          Bonding
        </Typography>
      </Box>
      <Box sx={{position: 'relative'}}>
          <Box
              sx={{
                position: "absolute",
                zIndex: -2,
                bottom: '0',
                background: 'radial-gradient(closest-side at 50% 50%, #E72850 0%, #E7285000 100%)',
                transform: 'matrix(-0.77, 0.64, -0.64, -0.77, 0, 0)',
                right: '70%',
                width: 1699,
                height: 1621,
                pointerEvents: 'none',
                opacity: 0.7,
              }}
            />   
          <Box
              sx={{
                position: "absolute",
                zIndex: -2,
                top: '0',
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
              background: `url(${shapes})`,
              backgroundSize: 'contain',
              top: -100,
              right: '85vw',
              width: 1012,
              height: 839,
              pointerEvents: 'none',
              display:{xs: 'none', md: 'block'},
            }}
          />
        <Box
            sx={{
              position: "absolute",
              zIndex: -1,
              background: `url(${shapes})`,
              backgroundSize: 'contain',
              bottom: '0',
              left: "80%",
              transform: 'scaleX(-1)',
              width: 1227,
              height: 1018,
              pointerEvents: 'none',
              display:{xs: 'none', md: 'block'},
            }}
          />  

        <Box sx={{ m: "auto", maxWidth: 1164, px: 3.75, mt: 15.5 }}>
          <Box
            sx={{
              display: "flex",
              flexWrap: "wrap",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            <Box sx={{ flexBasis: { xs: "100%", md: "50%" }, flexGrow: 1 }}>
              <Box sx={{ px: { xs: 0, md: 4 } }}>
                <Box sx={{ maxWidth: 460, mx: "auto" }}>
                  <Typography variant="h3">
                    <Box component="span" sx={{ color: "#FFCE00" }}>
                      Integrate into{" "}
                    </Box>
                    <Box component="span" sx={{ display: "inline-block" }}>
                      every chain at once
                    </Box>
                  </Typography>
                  <Typography sx={{ mt: 2 }}>
                    Wormhole SDK integrates your project with our generic
                    messaging layer. Wormhole SDK makes it easier than ever for
                    teams, apps, protocols, and users to move value seamlessly
                    across networks without fees.
                  </Typography>
                </Box>
              </Box>
            </Box>
            <Box
              sx={{
                mt: { xs: 8, md: 0 },
                flexBasis: { xs: "100%", md: "50%" },
                textAlign: "center",
                flexGrow: 1,
                backgroundColor: "rgba(255,255,255,.06)",
                backdropFilter: "blur(3px)",
                borderRadius: "37px",
                pt: 9.75,
                pb: 9,
              }}
            >
              <img src={stack} alt="" />
            </Box>
          </Box>
          <Box
            sx={{
              display: "flex",
              flexWrap: "wrap-reverse",
              alignItems: "center",
              justifyContent: "center",
              mt: 12.75,
            }}
          >
            <Box
              sx={{
                mt: { xs: 8, md: 0 },
                flexBasis: { xs: "100%", md: "50%" },
                textAlign: "center",
                flexGrow: 1,
                backgroundColor: "rgba(255,255,255,.06)",
                backdropFilter: "blur(3px)",
                borderRadius: "37px",
                pt: { xs: 3, md: 9.75 },
                pb: { xs: 3, md: 9 },
                px: { xs: 3, md: 8 },
              }}
            >
              <img src={network} alt="" style={{ maxWidth: "100%" }} />
            </Box>
            <Box sx={{ flexBasis: { xs: "100%", md: "50%" }, flexGrow: 1 }}>
              <Box sx={{ px: { xs: 0, md: 4 } }}>
                <Box sx={{ maxWidth: 460, mx: "auto" }}>
                  <Typography variant="h3">
                    <Box component="span" sx={{ color: "#FFCE00" }}>
                      Connecting projects{" "}
                    </Box>
                    <Box component="span" sx={{ display: "inline-block" }}>
                      to networks
                    </Box>
                  </Typography>
                  <Typography sx={{ mt: 2 }}>
                    Six high-value networks, two centralized exchanges, and 19 dexes.
                    Anyone in the community can add new networks to the protocol
                    and build the future of blockchain.
                  </Typography>
                </Box>
              </Box>
            </Box>
          </Box>
          <Box
            sx={{
              display: "flex",
              flexWrap: "wrap",
              alignItems: "center",
              justifyContent: "center",
              mt: 12.75,
            }}
          >
            <Box sx={{ flexBasis: { xs: "100%", md: "50%" }, flexGrow: 1 }}>
              <Box sx={{ px: { xs: 0, md: 4 } }}>
                <Box sx={{ maxWidth: 460, mx: "auto" }}>
                  <Typography variant="h3">
                    <Box component="span" sx={{ color: "#FFCE00" }}>
                      <AvoidBreak spans={["A trust-minimized"]} />
                    </Box>
                    <Box component="span" sx={{ display: "inline-block" }}>
                      build
                    </Box>
                  </Typography>
                  <Typography sx={{ mt: 2 }}>
                    Wormhole is built to be trust-minimized from the ground up
                    with a group of six networks secured by 19 equally weighted
                    guardians in the core layer.
                  </Typography>
                </Box>
              </Box>
            </Box>
            <Box
              sx={{
                mt: { xs: 8, md: 0 },
                flexBasis: { xs: "100%", md: "50%" },
                textAlign: "center",
                flexGrow: 1,
                backgroundColor: "rgba(255,255,255,.06)",
                backdropFilter: "blur(3px)",
                borderRadius: "37px",
                pt: 9.75,
                pb: 9,
                pl: 8,
                pr: 4,
              }}
            >
              <Timeline
                sx={{
                  p: 0,
                  m: 0,
                  "& .MuiTimelineItem-root": { minHeight: 52 },
                  "& .MuiTimelineItem-root:last-child": { minHeight: 0 },
                  "& .MuiTimelineItem-root:before": { display: "none" },
                  "& .MuiTimelineConnector-root": {
                    width: "1px",
                    my: 0.25,
                    backgroundColor: "transparent",
                    borderLeft: "1px dashed #bdbdbd",
                  },
                }}
              >
                <TimelineItem>
                  <TimelineSeparator>
                    <img src={shape} alt="" />
                    <TimelineConnector />
                  </TimelineSeparator>
                  <TimelineContent>Send your message to Wormhole</TimelineContent>
                </TimelineItem>
                <TimelineItem>
                  <TimelineSeparator>
                    <img src={shape} alt="" />
                    <TimelineConnector />
                  </TimelineSeparator>
                  <TimelineContent>
                    The Guardian network observes the transaction
                  </TimelineContent>
                </TimelineItem>
                <TimelineItem>
                  <TimelineSeparator>
                    <img src={shape} alt="" />
                    <TimelineConnector />
                  </TimelineSeparator>
                  <TimelineContent>Quorum is achieved in seconds</TimelineContent>
                </TimelineItem>
                <TimelineItem>
                  <TimelineSeparator>
                    <img src={shape} alt="" />
                    <TimelineConnector />
                  </TimelineSeparator>
                  <TimelineContent>
                    Guardians make your attested message publicly available.
                  </TimelineContent>
                </TimelineItem>
                <TimelineItem>
                  <TimelineSeparator>
                    <img src={shape} alt="" />
                  </TimelineSeparator>
                  <TimelineContent>
                    Access your message on a different chain
                  </TimelineContent>
                </TimelineItem>
              </Timeline>
            </Box>
          </Box>
        </Box>
      </Box>
    </Layout>
  );
};

export default BuidlPage;
