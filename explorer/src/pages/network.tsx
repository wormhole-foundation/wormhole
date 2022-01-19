import {
  Box,
  CircularProgress,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from "@mui/material";
import * as React from "react";
import HeroText from "../components/HeroText";
import Layout from "../components/Layout";
import shape1 from "../images/index/shape1.svg";
import { Heartbeat } from "@certusone/wormhole-sdk/lib/esm/proto/gossip/v1/gossip";
import {
  GrpcWebImpl,
  PublicRPCServiceClientImpl,
} from "@certusone/wormhole-sdk/lib/esm/proto/publicrpc/v1/publicrpc";
import ReactTimeAgo from "react-time-ago";

// TODO: network switcher
const activeNetwork = { name: "mainnet" };

const GuardiansList = () => {
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
      "https://wormhole-v2-mainnet-api.certus.one",
      {}
    );
    const publicRpc = new PublicRPCServiceClientImpl(rpc);
    const interval = setInterval(() => {
      (async () => {
        try {
          const response = await publicRpc.GetLastHeartbeats({});
          response.entries.map((entry) =>
            entry.rawHeartbeat
              ? addHeartbeat(activeNetwork.name, entry.rawHeartbeat)
              : null
          );
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
      <Box sx={{ px: 4 }}>
        <Typography variant="h5">
          {foundHeartbeats
            ? `${guardianCount} Guardian${
                guardianCount > 1 ? "s" : ""
              } currently broadcasting`
            : `Listening for Guardian heartbeats...`}
        </Typography>
      </Box>
      <Box
        sx={{
          backgroundColor: "rgba(255,255,255,.07)",
          borderRadius: "28px",
          mt: 5,
          p: 4,
        }}
      >
        {foundHeartbeats ? (
          <TableContainer>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Guardian</TableCell>
                  <TableCell>Version</TableCell>
                  <TableCell>Networks</TableCell>
                  <TableCell>Heartbeat</TableCell>
                  <TableCell>Last Heartbeat</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {sortedHeartbeats.map((hb) => (
                  <TableRow key={hb.nodeName}>
                    <TableCell>
                      {hb.nodeName}
                      <br />
                      {hb.guardianAddr}
                    </TableCell>
                    <TableCell sx={{ whiteSpace: "nowrap" }}>
                      {hb.version}
                    </TableCell>
                    <TableCell sx={{ whiteSpace: "nowrap" }}>
                      {hb.networks.map((network) => network.id)}
                    </TableCell>
                    <TableCell>{hb.counter}</TableCell>
                    <TableCell sx={{ "& > time": { whiteSpace: "nowrap" } }}>
                      <ReactTimeAgo
                        date={new Date(Number(hb.timestamp.slice(0, -6)))}
                        timeStyle="round"
                      />
                    </TableCell>
                  </TableRow>
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
        <HeroText
          heroSpans={["Meet the Guardians"]}
          subtitleText={[
            "The 19 guardians in Wormhole's guardian network each hold",
            "equal weight in governance consensus.",
          ]}
        />
      </Box>
      <Box sx={{ maxWidth: 1220, mx: "auto", mt: 30, px: 3.75 }}>
        <GuardiansList />
      </Box>
    </Layout>
  );
};

export default AppsPage;
