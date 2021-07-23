import {
  AppBar,
  Button,
  Grid,
  Link,
  makeStyles,
  MenuItem,
  TextField,
  Toolbar,
  Typography,
} from "@material-ui/core";
import { useCallback } from "react";
import EthereumSignerKey from "./components/EthereumSignerKey";
import SolanaWalletKey from "./components/SolanaWalletKey";
import { useEthereumProvider } from "./contexts/EthereumProviderContext";
import { Bridge__factory } from "./ethers-contracts";
import wormholeLogo from "./icons/wormhole.svg";
import { ETH_TOKEN_BRIDGE_ADDRESS } from "./utils/consts";

const useStyles = makeStyles((theme) => ({
  appBar: {
    borderBottom: `.5px solid ${theme.palette.divider}`,
    "& > .MuiToolbar-root": {
      height: 82,
      margin: "auto",
      maxWidth: 1100,
    },
  },
  spacer: {
    flex: 1,
    width: "100vw",
  },
  link: {
    color: theme.palette.text.primary,
    marginLeft: theme.spacing(6),
  },
  sideBar: {
    position: "fixed",
    top: 0,
    left: 0,
    height: 733,
    maxHeight: "80vh",
    width: 50,
    borderRight: `.5px solid ${theme.palette.divider}`,
    borderBottom: `.5px solid ${theme.palette.divider}`,
  },
  content: {
    margin: theme.spacing(10.5, 8),
  },
  transferBox: {
    width: 540,
    margin: "auto",
    border: `.5px solid ${theme.palette.divider}`,
    padding: theme.spacing(5.5, 12),
  },
  arrow: {
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
  },
  transferField: {
    marginTop: theme.spacing(5),
  },
  transferButton: {
    marginTop: theme.spacing(7.5),
    textTransform: "none",
    width: "100%",
  },
}));

function App() {
  const classes = useStyles();
  const provider = useEthereumProvider();
  const handleClick = useCallback(() => {
    const bridge = Bridge__factory.connect(ETH_TOKEN_BRIDGE_ADDRESS, provider);
    bridge.chainId().then((n) => console.log(n));
  }, [provider]);
  return (
    <>
      <AppBar position="static" color="inherit" className={classes.appBar}>
        <Toolbar>
          <img src={wormholeLogo} alt="Wormhole Logo" />
          <div className={classes.spacer} />
          <Link className={classes.link}>Placeholder</Link>
          <Link className={classes.link}>Placeholder</Link>
          <Link className={classes.link}>Placeholder</Link>
        </Toolbar>
      </AppBar>
      <div className={classes.sideBar}></div>
      <div className={classes.content}>
        <div className={classes.transferBox}>
          <Grid container>
            <Grid item xs={4}>
              <Typography>To</Typography>
              <TextField select fullWidth value="ETH">
                <MenuItem value="ETH">Ethereum</MenuItem>
                <MenuItem value="SOL">Solana</MenuItem>
              </TextField>
              <EthereumSignerKey />
            </Grid>
            <Grid item xs={4} className={classes.arrow}>
              &rarr;
            </Grid>
            <Grid item xs={4}>
              <Typography>From</Typography>
              <TextField select fullWidth value="SOL">
                <MenuItem value="ETH">Ethereum</MenuItem>
                <MenuItem value="SOL">Solana</MenuItem>
              </TextField>
              <SolanaWalletKey />
            </Grid>
          </Grid>
          <TextField
            placeholder="Asset"
            fullWidth
            className={classes.transferField}
          />
          <TextField
            placeholder="Amount"
            type="number"
            fullWidth
            className={classes.transferField}
          />
          <Button
            color="primary"
            variant="contained"
            className={classes.transferButton}
            onClick={handleClick}
          >
            Transfer
          </Button>
        </div>
      </div>
    </>
  );
}

export default App;
