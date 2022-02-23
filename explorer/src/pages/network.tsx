import {
  Box,
  CircularProgress,
  Collapse,
  IconButton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from "@mui/material";
import * as React from "react";
import { PageProps } from 'gatsby'
import HeroText from "../components/HeroText";
import Layout from "../components/Layout";
import shape1 from "../images/index/shape1.svg";
import { Heartbeat } from "@certusone/wormhole-sdk/lib/esm/proto/gossip/v1/gossip";
import {
  GrpcWebImpl,
  PublicRPCServiceClientImpl,
} from "@certusone/wormhole-sdk/lib/esm/proto/publicrpc/v1/publicrpc";
import ReactTimeAgo from "react-time-ago";
import NetworkSelect from "../components/NetworkSelect";
import { useNetworkContext } from "../contexts/NetworkContext";
import ChainIcon from "../components/ChainIcon";
import { ChainId } from "@certusone/wormhole-sdk";
import { KeyboardArrowDown, KeyboardArrowUp } from "@mui/icons-material";
import { ChainID } from "../utils/consts";
import { SEO } from "../components/SEO";
import shapes from "../images/shape.png";

const GuardianRow = ({ hb }: { hb: Heartbeat }) => {
  const [open, setOpen] = React.useState(false);
  return (
    <>
      <TableRow>
        <TableCell sx={{ p: 0 }}>
          <IconButton
            aria-label="expand row"
            size="small"
            onClick={() => setOpen(!open)}
          >
            {open ? <KeyboardArrowUp /> : <KeyboardArrowDown />}
          </IconButton>
        </TableCell>
        <TableCell>
          {hb.nodeName}
          <br />
          {hb.guardianAddr}
        </TableCell>
        <TableCell sx={{ whiteSpace: "nowrap" }}>{hb.version}</TableCell>
        <TableCell sx={{ whiteSpace: "nowrap" }}>
          <Box sx={{ display: "flex", alignItems: "center" }}>
            {hb.networks.map((network) => (
              <ChainIcon key={network.id} chainId={network.id as ChainId} />
            ))}
          </Box>
        </TableCell>
        <TableCell align="right">{hb.counter}</TableCell>
        <TableCell sx={{ "& > time": { whiteSpace: "nowrap" } }}>
          <ReactTimeAgo
            date={new Date(Number(hb.timestamp.slice(0, -6)))}
            timeStyle="round"
          />
        </TableCell>
      </TableRow>
      <TableRow>
        <TableCell sx={{ py: 0 }} colSpan={6}>
          <Collapse in={open} timeout="auto" unmountOnExit>
            <TableContainer>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell></TableCell>
                    <TableCell>Network</TableCell>
                    <TableCell>Contract Address</TableCell>
                    <TableCell align="right">Block Height</TableCell>
                    <TableCell align="right">Error Count</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {hb.networks.map((n) => (
                    <TableRow key={n.id}>
                      <TableCell component="th" scope="row">
                        <ChainIcon chainId={n.id as ChainId} />
                      </TableCell>
                      <TableCell>{ChainID[n.id]}</TableCell>
                      <TableCell>{n.contractAddress}</TableCell>
                      <TableCell align="right">{n.height}</TableCell>
                      <TableCell align="right">{n.errorCount}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </Collapse>
        </TableCell>
      </TableRow>
    </>
  );
};

const GuardiansList = () => {
  const { activeNetwork } = useNetworkContext();
  const [heartbeats, setHeartbeats] = React.useState<{
    [networkName: string]: { [nodeName: string]: Heartbeat };
  }>({ devnet: {}, testnet: {}, mainnet: {} });

  // TODO: add all heartbeats at once
  const addHeartbeat = React.useCallback(
    (networkName: string, hbObj: Heartbeat) => {
      hbObj.networks.sort((a: any, b: any) => a.id - b.id);
      const { nodeName } = hbObj;
      setHeartbeats((heartbeats) => ({
        ...heartbeats,
        [networkName]: { ...heartbeats[networkName], [nodeName]: hbObj },
      }));
    },
    []
  );

  React.useEffect(() => {
    let cancelled = false;
    const rpc = new GrpcWebImpl(
      String(activeNetwork.endpoints.guardianRpcBase),
      {}
    );
    const publicRpc = new PublicRPCServiceClientImpl(rpc);
    const interval = setInterval(() => {
      (async () => {
        try {
          const response = await publicRpc.GetLastHeartbeats({});
          if (!cancelled) {
            response.entries.map((entry) =>
              entry.rawHeartbeat
                ? addHeartbeat(activeNetwork.name, entry.rawHeartbeat)
                : null
            );
          }
        } catch (e) {
          console.error("GetLastHeartbeats error:", e);
        }
      })();
    }, 3000);
    return () => {
      clearInterval(interval);
      cancelled = true;
    };
  });
  const activeHeartbeats = heartbeats[activeNetwork.name];
  const guardianCount = Object.keys(activeHeartbeats).length;
  const foundHeartbeats = guardianCount > 0;
  const sortedHeartbeats = React.useMemo(() => {
    const arr = [...Object.values(activeHeartbeats)];
    arr.sort((a, b) => a.nodeName.localeCompare(b.nodeName));
    return arr;
  }, [activeHeartbeats]);
  return (
    <>
      <Box
        sx={{ px: 4, display: "flex", flexWrap: "wrap", alignItems: "center" }}
      >
        <Typography variant="h5">
          {foundHeartbeats
            ? `${guardianCount} Guardian${
                guardianCount > 1 ? "s" : ""
              } currently broadcasting`
            : `Listening for Guardian heartbeats...`}
        </Typography>
        <Box sx={{ flexGrow: 1 }} />
        <NetworkSelect />
      </Box>
      <Box
        sx={{
          backgroundColor: "rgba(255,255,255,.07)",
          borderRadius: "28px",
          mt: 4,
          p: 4,
        }}
      >
        {foundHeartbeats ? (
          <TableContainer>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell></TableCell>
                  <TableCell>Guardian</TableCell>
                  <TableCell>Version</TableCell>
                  <TableCell>Networks</TableCell>
                  <TableCell align="right">Heartbeat</TableCell>
                  <TableCell>Last Heartbeat</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {sortedHeartbeats.map((hb) => (
                  <GuardianRow key={hb.nodeName} hb={hb} />
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        ) : (
          <Box sx={{ textAlign: "center" }}>
            <CircularProgress />
          </Box>
        )}
      </Box>
    </>
  );
};

const NetworkPage = ({ location }: PageProps) => {
  return (
    <Layout>
      <SEO
        title="Network"
        description="Meet the Guardians."
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
          heroSpans={["Meet the Guardians"]}
          subtitleText={[
            "The 19 guardians in Wormhole's guardian network each hold",
            "equal weight in governance consensus.",
          ]}
        />
      </Box>
      <Box sx={{position: 'relative'}}>
        <Box
            sx={{
              position: "absolute",
              zIndex: -2,
              top: '0',
              background: 'radial-gradient(closest-side at 50% 50%, #5189C8 0%, #5189C800 100%) ',
              transform: 'matrix(-0.67, 0.74, -0.74, -0.67, 0, 0)',
              left: '70%',
              width: 1077,
              height: 1329,
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
                top: '0',
                left: "85%",
                transform: 'scaleX(-1)',
                width: 1227,
                height: 1018,
                pointerEvents: 'none',
                display:{xs: 'none', md: 'block'},
              }}
            />  
      
      <Box sx={{ maxWidth: 1220, mx: "auto", mt: 30, px: 3.75 }}>
        <GuardiansList />
      </Box>
      </Box>
    </Layout>
  );
};

export default NetworkPage;
