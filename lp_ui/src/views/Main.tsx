import {
  Container,
  makeStyles,
  Typography,
  Paper,
  Tab,
} from "@material-ui/core";
import { useCallback, useState } from "react";
import LogWatcher from "../components/LogWatcher";
import SolanaWalletKey from "../components/SolanaWalletKey";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";
import TabContext from "@material-ui/lab/TabContext";
import TabList from "@material-ui/lab/TabList";
import TabPanel from "@material-ui/lab/TabPanel";

const useStyles = makeStyles(() => ({}));

function Main() {
  const classes = useStyles();
  const wallet = useSolanaWallet();
  const [selectedTab, setSelectedTab] = useState("createPool");
  const handleChange = useCallback(
    (event, value) => {
      setSelectedTab(value);
    },
    [setSelectedTab]
  );

  const content = !wallet.publicKey ? (
    <Typography>Please connect your wallet.</Typography>
  ) : (
    <TabContext value={selectedTab}>
      <TabList onChange={handleChange} aria-label="simple tabs example">
        <Tab label="Create Pool" value="createPool" />
        <Tab label="Add Liquidity" value="Add Liquidity" />
        <Tab label="Redeem Liquidity" value="Redeem Liquidity" />
      </TabList>
      <TabPanel value="1">Item One</TabPanel>
      <TabPanel value="2">Item Two</TabPanel>
      <TabPanel value="3">Item Three</TabPanel>
    </TabContext>
  );

  return (
    <Container maxWidth="md">
      <Paper style={{ padding: "3rem" }}>
        <SolanaWalletKey />
        {content}
      </Paper>
      <LogWatcher />
    </Container>
  );
}

export default Main;
