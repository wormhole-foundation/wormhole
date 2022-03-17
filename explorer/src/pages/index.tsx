import ArrowForward from "@mui/icons-material/ArrowForward";
import { Box, Button, Grid, Typography } from "@mui/material";
import {
  Link as RouterLink,
  PageProps,
  graphql,
  useStaticQuery
} from "gatsby";
import React, { useEffect, useState } from "react";
import GridWithCards from "../components/GridWithCards";
import HeroText from "../components/HeroText";
import Layout from "../components/Layout";
import { SEO } from "../components/SEO";
import { portal as portalUrl } from "../utils/urls";

import { gsap } from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";

import apps from "../images/index/apps2.png";
import blob from "../images/index/blob.svg";
import cross from "../images/index/cross.svg";
import cube from "../images/index/cube.svg";
import portal from "../images/index/portal.png";
import protocols from "../images/index/protocol_new.png";
import shape1 from "../images/index/shape1.svg";
import shape from "../images/shape.png";
import shape2 from "../images/shape2.png";
import { Totals, NotionalTvl } from "../components/ExplorerStats/ExplorerStats";
import { amountFormatter } from "../utils/explorer";

const featuredNumber = { fontSize: 42, fontFamily: "Suisse BP Intl", fontWeight: "bold" };
const statsBaseUrl = "https://europe-west3-wormhole-315720.cloudfunctions.net/mainnet-"

const IndexPage = ({ location }: PageProps) => {
  const { site } = useStaticQuery<IndexQueryType>(IndexStaticQuery)
  const [tvl, setTvl] = useState<number | undefined>(undefined)
  const [messageTotal, setMessageTotal] = useState<number | undefined>(undefined)

  let statsInterval: NodeJS.Timer | undefined = undefined
  const controller = new AbortController()
  const { signal } = controller


  const logo = {
    "@type": "ImageObject",
    "url": `${site.siteMetadata.siteUrl}/logo-and-name-stacked.png`,
    "height": "146",
    "width": "146"
  }
  const structuredData = {
    "@context": "https://schema.org",
    "@type": "Organization",
    "@id": "https://wormholenetwork.com#organization",
    mainEntityOfPage: "https://wormholenetwork.com#organization",
    url: "https://wormholenetwork.com",
    name: "Wormhole",
    sameAs: [
      "https://github.com/certusone/wormhole",
      "https://t.me/wormholecrypto",
      "https://twitter.com/wormholecrypto",
      "https://wormholebridge.com",
      "https://wormholecrypto.medium.com",
      "https://discord.gg/wormholecrypto",
    ],
    alternateName: [
      "wormhole network",
      "wormhole protocol",
      "wormhole bridge",
      "wormhole crypto",
      "certus one wormhole",
      "solana wormhole",
      "SOL wormhole",
      "terra wormhole",
      "LUNA wormhole",
      "ethereum wormhole",
      "ETH wormhole",
      "binance wormhole",
      "BSC wormhole",
      "oasis wormhole",
      "ROSE wormhole",
      "avalanche wormhole",
      "AVAX wormhole"
    ],
    description: "A cross-chain messaging protocol.",
    image: logo,
    logo: logo
  }

  const headerImage = React.useRef<HTMLCanvasElement>(null);
  const gradient1 = React.useRef<HTMLCanvasElement>(null);
  const gradient2 = React.useRef<HTMLCanvasElement>(null);


  function fetchStats() {
    const tvlUrl = `${statsBaseUrl}notionaltvl`
    const messagesUrl = `${statsBaseUrl}totals`

    fetch(tvlUrl, { signal }).then((res) => {
      if (res.ok) return res.json();
    }).then((result: NotionalTvl) => {
      setTvl(result.AllTime["*"]["*"].Notional);
    }, (error) => {
      if (error.name !== "AbortError") console.error("failed fetching notional TVL. error: ", error);
    });
    fetch(messagesUrl, { signal }).then((res) => {
      if (res.ok) return res.json();
    }).then((result: Totals) => {
      setMessageTotal(result.TotalCount["*"]);
    }, (error) => {
      if (error.name !== "AbortError") console.error("failed fetching totals. error: ", error);
    });
  }

  useEffect(() => {
    fetchStats()  // fetchStats on first load
    statsInterval = setInterval(fetchStats, 30000) // fetch every 30 seconds

    gsap.registerPlugin(ScrollTrigger);


    gsap.from(headerImage.current, {
      scale: 1.1,
      duration: 10,
      delay: 1,
      rotation: 3,
      ease: "Power3.easeOut",
    })

    gsap.to(gradient1.current, {
      scale: 1.2,
      ease: "Power3.easeOut",
      x: 300,
      scrollTrigger: {
        trigger: gradient1.current,
        start: "-0% 0%",
        end: "+=500",
        scrub: 1,
      },
    })

    gsap.from(gradient2.current, {
      scale: 0.5,
      ease: "Power3.easeOut",
      scrollTrigger: {
        trigger: gradient2.current,
        start: "-50% 50%",
        end: "+=1000",
        scrub: 1,
      },
    })

    return function cleanup() {
      // clear any ongoing intervals
      if (statsInterval) {
        clearInterval(statsInterval);
      }
      // abort any in-flight requests
      controller.abort();
    }
  }, [])

  return (
    <Layout>
      <SEO
        // use default title for index
        description="The best of blockchains. Move information and value anywhere."
        pathname={location.pathname}
      >
        <script type="application/ld+json">
          {JSON.stringify(structuredData, undefined, 4)}
        </script>
      </SEO>
      <Box sx={{ position: "relative", marginTop: 21 }}>
        <Box
          ref={headerImage}
          sx={{
            position: "absolute",
            zIndex: -1,
            transform: "translate(0px, -25%)",
            background: `url(${shape1})`,
            backgroundRepeat: "no-repeat",
            backgroundPosition: "top -240px center",
            backgroundSize: "2070px 1155px",
            width: "100%",
            height: 1155,
          }}
        />
        <HeroText
          maxWidth={600}
          heroSpans={["The best of", "blockchains"]}
          subtitleText="Move information and value anywhere."
        />
        <Box
          sx={{
            m: "auto",
            maxWidth: { xs: 240, sm: 600 },
            textAlign: "center",
          }}
        >
          <Box
            sx={{
              width: "calc( 100% - 16px )",
              mx: "auto",
              mt: 15.5,
              display: "flex",
              flexWrap: "wrap",
              justifyContent: "center",
            }}
          >
            {tvl && <Box
              sx={{
                mt: 2,
                mx: 1,
                pt: { xs: 1, sm: 0 },
                display: "flex",
                alignItems: "center",
                justifyContent: "space-evenly",
                flexBasis: { xs: "100%", sm: "calc(33.33333% - 16px)" },
                borderTop: "1px solid white",
              }}
            >
              <Typography sx={featuredNumber}>${amountFormatter(tvl, 2)}</Typography>
              <Typography variant="body2">in TVL</Typography>
            </Box>}
            <Box
              sx={{
                mt: 2,
                mx: 1,
                pt: { xs: 1, sm: 0 },
                display: "flex",
                alignItems: "center",
                justifyContent: "space-evenly",
                flexBasis: { xs: "100%", sm: "calc(33.33333% - 16px)" },
                borderTop: "1px solid white",
              }}
            >
              <Typography sx={featuredNumber}>7</Typography>
              <Typography variant="body2">chain integrations</Typography>
            </Box>
            {messageTotal && <Box
              sx={{
                mt: 2,
                mx: 1,
                pt: { xs: 1, sm: 0 },
                display: "flex",
                alignItems: "center",
                justifyContent: "space-evenly",
                flexBasis: { xs: "100%", sm: "calc(33.33333% - 16px)" },
                borderTop: "1px solid white",
              }}
            >
              <Typography sx={featuredNumber}>
                {amountFormatter(messageTotal, 0)}
              </Typography>
              <Typography variant="body2">txs</Typography>
            </Box>}
          </Box>
        </Box>
      </Box>


      <Box sx={{ position: 'relative' }}>
        <Box
          ref={gradient1}
          sx={{
            position: "absolute",
            zIndex: -1,
            top: '50%',
            background: 'radial-gradient(closest-side at 50% 50%, #E72850 0%, #E7285000 100%)',
            transform: 'matrix(0.96, 0.29, -0.29, 0.96, 0, 0)',
            left: '60%',
            width: 1645,
            height: 903,
            pointerEvents: 'none',
            opacity: 0.7,
          }}
        />
        <Box
          ref={gradient2}
          sx={{
            position: "absolute",
            zIndex: -1,
            top: '65%',
            background: 'radial-gradient(closest-side at 50% 50%, #5189C8 0%, #5189C800 100%) ',
            transform: 'matrix(0.67, 0.74, -0.74, 0.67, 0, 0)',
            left: '5%',
            width: 1136,
            height: 1489,
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
            right: '70vw',
            width: 1363,
            height: 1130,
            pointerEvents: 'none',
            display: { xs: 'none', md: 'block' },
          }}
        />
        <Box
          sx={{
            display: "flex",
            flexWrap: "wrap",
            maxWidth: 1220,
            px: 3.75,
            mt: 35,
            mx: "auto",
            alignItems: "center",
            justifyContent: "center",
          }}
        >

          <Box
            sx={{
              flexBasis: { xs: "100%", md: "40%" },
              flexGrow: 1,
            }}
          >
            <Box sx={{ px: { xs: 0, md: 4 } }}>
              <Box sx={{ maxWidth: 348, mx: "auto" }}>
                <Typography variant="h3">
                  <Box component="span" sx={{ color: "#FFCE00" }}>
                    Protocol:{" "}
                  </Box>
                  <Box component="span" sx={{ display: "inline-block" }}>
                    the core layer
                  </Box>
                </Typography>
                <Typography sx={{ mt: 2 }}>
                  The foundation that an ecosystem of apps is built on top of.
                </Typography>
                <Button
                  component={RouterLink}
                  to="/buidl/"
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
              mt: { xs: 8, md: null },
              flexBasis: { xs: "100%", md: "60%" },
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
          <Box sx={{ px: { xs: 0, md: 4 } }}>
            <Box sx={{ maxWidth: 348, mx: "auto" }}>
              <Typography variant="h3">
                <Box component="span" sx={{ color: "#FFCE00" }}>
                  Apps:{" "}
                </Box>
                <Box component="span">endless possibilities</Box>
              </Typography>
              <Typography sx={{ mt: 2 }}>
                Apps can now live across chains at once and integrate the best
                of each.
              </Typography>
              <Button
                component={RouterLink}
                to="/apps/"
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
      </Box>

      <Box sx={{ position: 'relative' }}>
        <Box
          sx={{
            position: "absolute",
            zIndex: -1,
            background: `url(${shape2})`,
            backgroundSize: 'contain',
            top: '50%',
            transform: 'translateY(-50%)',
            left: '75vw',
            width: 1612,
            height: 1316,
            pointerEvents: 'none',
            display: { xs: 'none', md: 'block' },
          }}
        />
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
              flexBasis: { xs: "100%", md: "40%" },
              flexGrow: 1,
            }}
          >
            <Box sx={{ px: { xs: 0, md: 4 } }}>
              <Box sx={{ maxWidth: 340, mx: "auto" }}>
                <Typography variant="h3">
                  <Box component="span" sx={{ color: "#FFCE00" }}>
                    Portal:{" "}
                  </Box>
                  <Box component="span" sx={{ display: "inline-block" }}>
                    a token bridge
                  </Box>
                </Typography>
                <Typography sx={{ mt: 2 }}>
                  Never have to retrace your steps, with unlimited transfers
                  across chains for tokens and NFTs wrapped by Wormhole.
                </Typography>
                <Button
                  href={portalUrl}
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
              mt: { xs: 8, md: null },
              flexBasis: { xs: "100%", md: "60%" },
              textAlign: "center",
              flexGrow: 1,
            }}
          >
            <Button variant="text" href={portalUrl} sx={{
              '&:hover': {
                background: 'transparent'
              },
              'span': {
                display: 'none'
              }
            }}>
              <Box component="img" src={portal} alt="" sx={{ maxWidth: "100%" }} />
            </Button>
          </Box>
        </Box>
      </Box>

      <Box sx={{ textAlign: "center", mt: 12.5, px: 2 }}>
        <Typography variant="h3">
          <Box component="span" sx={{ color: "#FFCE00" }}>
            Cross-chain
          </Box>
          <Box component="span"> everything</Box>
        </Typography>
        <Typography sx={{ mt: 2, maxWidth: 480, mx: "auto", fontWeight: 300 }}>
          Each blockchain has a distinct strength. Wormhole lets you get the
          best out of every blockchain without compromise.
        </Typography>
      </Box>
      <Box sx={{ maxWidth: 1220, mx: "auto", mt: 12, px: 3.75 }}>
        <GridWithCards
          data={[
            {
              src: cross,
              size: 220,
              header: "Never stop expanding",
              description:
                "Chains, information, and users are growing everyday. Build on a protocol that is set up to scale, with no limits, right from the start.",
            },
            {
              src: blob,
              size: 220,
              header: "Explore and experiment",
              description:
                "Now is the time to explore and experiment. The only limit to what you're able to build is your imagination.",
            },
            {
              src: cube,
              size: 220,
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

type IndexQueryType = {
  site: {
    siteMetadata: {
      siteUrl: string
    }
  }
}
const IndexStaticQuery = graphql`
  query Index {
    site {
      siteMetadata {
        siteUrl
      }
    }
  }
`

export default IndexPage;
