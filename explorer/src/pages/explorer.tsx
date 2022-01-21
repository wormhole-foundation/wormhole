import { CloseOutlined } from "@mui/icons-material";
import { Box, Button, Typography } from "@mui/material";
import { Link as RouterLink, PageProps } from "gatsby";
import { OutboundLink } from "gatsby-plugin-google-gtag";
import * as React from "react";
import ExplorerStats from "../components/ExplorerStats/ExplorerStats";
import HeroText from "../components/HeroText";
import Layout from "../components/Layout";
import NetworkSelect from "../components/NetworkSelect";
import shape1 from "../images/index/shape1.svg";
import { ChainID } from "../utils/consts";
import {
  contractNameFormatter,
  nativeExplorerContractUri,
} from "../utils/explorer";
import { explorer } from "../utils/urls";

// form props
interface ExplorerQueryValues {
  emitterChain: number;
  emitterAddress: string;
  sequence: string;
  txId: string;
}

const ExplorerPage = ({ location, navigate }: PageProps) => {
  const [emitterChain, setEmitterChain] =
    React.useState<ExplorerQueryValues["emitterChain"]>();
  const [emitterAddress, setEmitterAddress] =
    React.useState<ExplorerQueryValues["emitterAddress"]>();
  const [sequence, setSequence] =
    React.useState<ExplorerQueryValues["sequence"]>();
  const [txId, setTxId] = React.useState<ExplorerQueryValues["txId"]>();
  const [showQueryForm, setShowQueryForm] = React.useState<boolean>(false);
  const [doneReadingQueryParams, setDoneReadingQueryParams] =
    React.useState<boolean>(false);

  React.useEffect(() => {
    if (location.search) {
      // take searchparams from the URL and set the values in the form
      const searchParams = new URLSearchParams(location.search);

      const chain = searchParams.get("emitterChain");
      const address = searchParams.get("emitterAddress");
      const seq = searchParams.get("sequence");
      const tx = searchParams.get("txId");

      // if the search params are different form values, update state
      if (Number(chain) !== emitterChain) {
        setEmitterChain(Number(chain) || undefined);
      }
      if (address !== emitterAddress) {
        setEmitterAddress(address || undefined);
      }
      if (seq !== sequence) {
        setSequence(seq || undefined);
      }
      if (tx !== txId) {
        setTxId(tx || undefined);
      }
      if (!tx && chain && address && seq) {
        setShowQueryForm(true);
      }
    } else {
      // clear state
      setEmitterChain(undefined);
      setEmitterAddress(undefined);
      setSequence(undefined);
      setTxId(undefined);
      setShowQueryForm(false);
    }
    // be explicit about when it is ok to render
    setDoneReadingQueryParams(true);
  }, [location.search]);

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
        <HeroText
          heroSpans={["Check the Stats"]}
          subtitleText={[
            "Explore real-time movement of information and",
            "value around the Wormhole ecosystem.",
          ]}
        />
      </Box>
      <Box sx={{ maxWidth: 1220, mx: "auto", mt: 30, px: 3.75 }}>
        <Box
          sx={{
            px: 4,
            display: "flex",
            flexWrap: "wrap",
            alignItems: "center",
          }}
        >
          <Box sx={{ flexGrow: 1 }} />
          <NetworkSelect />
        </Box>
      </Box>
      {!(emitterChain && emitterAddress && sequence) && !txId ? (
        <Box sx={{ maxWidth: 1220, mx: "auto", px: 3.75 }}>
          <Box
            sx={{
              backgroundColor: "rgba(255,255,255,.07)",
              borderRadius: "28px",
              mt: 4,
              p: 4,
            }}
          >
            {emitterAddress && emitterChain ? (
              // show heading with the context of the address
              <Typography variant="h4">
                Recent messages from {ChainID[emitterChain]}&nbsp;
                {nativeExplorerContractUri(emitterChain, emitterAddress) ? (
                  <OutboundLink
                    href={nativeExplorerContractUri(
                      emitterChain,
                      emitterAddress
                    )}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    {contractNameFormatter(emitterAddress, emitterChain)}
                  </OutboundLink>
                ) : (
                  contractNameFormatter(emitterAddress, emitterChain)
                )}
                :
              </Typography>
            ) : emitterChain ? (
              // show heading with the context of the chain
              <Typography variant="h4">
                Recent {ChainID[emitterChain]} activity
              </Typography>
            ) : (
              // show heading for root view, all chains
              <>
                <Typography variant="h4" gutterBottom>
                  Recent messages
                </Typography>
                <Typography variant="body2">
                  From all chains and addresses
                </Typography>
              </>
            )}
            {emitterAddress || emitterChain ? (
              <Button
                component={RouterLink}
                to={explorer}
                endIcon={<CloseOutlined />}
              >
                Clear
              </Button>
            ) : null}
          </Box>
          {doneReadingQueryParams && (
            <ExplorerStats
              emitterChain={emitterChain}
              emitterAddress={emitterAddress}
            />
          )}
        </Box>
      ) : null}
    </Layout>
  );
};

export default ExplorerPage;
