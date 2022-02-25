import {
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
} from "@certusone/wormhole-sdk";
import {
  AppBar,
  Container,
  Hidden,
  IconButton,
  Link,
  makeStyles,
  Tab,
  Tabs,
  Toolbar,
  Tooltip,
  Typography,
} from "@material-ui/core";
import { HelpOutline } from "@material-ui/icons";
import { useCallback } from "react";
import { useHistory, useLocation } from "react-router";
import {
  Link as RouterLink,
  NavLink,
  Redirect,
  Route,
  Switch,
} from "react-router-dom";
import Attest from "./components/Attest";
import Footer from "./components/Footer";
import HeaderText from "./components/HeaderText";
import Migration from "./components/Migration";
import EvmQuickMigrate from "./components/Migration/EvmQuickMigrate";
import SolanaQuickMigrate from "./components/Migration/SolanaQuickMigrate";
import NFT from "./components/NFT";
import NFTOriginVerifier from "./components/NFTOriginVerifier";
import Recovery from "./components/Recovery";
import Stats from "./components/Stats";
import TokenOriginVerifier from "./components/TokenOriginVerifier";
import Transfer from "./components/Transfer";
import WithdrawTokensTerra from "./components/WithdrawTokensTerra";
import { useBetaContext } from "./contexts/BetaContext";
import Portal from "./icons/portal_logo.svg";
import Header from "./images/Header.png";
import { CLUSTER } from "./utils/consts";

const useStyles = makeStyles((theme) => ({
  appBar: {
    background: "transparent",
    marginTop: theme.spacing(2),
    "& > .MuiToolbar-root": {
      margin: "auto",
      width: "100%",
      maxWidth: 1440,
    },
  },
  spacer: {
    flex: 1,
    width: "100vw",
  },
  link: {
    ...theme.typography.body2,
    fontWeight: 600,
    color: "black",
    marginLeft: theme.spacing(4),
    textUnderlineOffset: "6px",
    [theme.breakpoints.down("sm")]: {
      marginLeft: theme.spacing(2.5),
    },
    [theme.breakpoints.down("xs")]: {
      marginLeft: theme.spacing(1),
    },
    "&.active": {
      textDecoration: "underline",
    },
  },
  bg: {
    // background:
    //   "linear-gradient(160deg, rgba(69,74,117,.1) 0%, rgba(138,146,178,.1) 33%, rgba(69,74,117,.1) 66%, rgba(98,104,143,.1) 100%), linear-gradient(45deg, rgba(153,69,255,.1) 0%, rgba(121,98,231,.1) 20%, rgba(0,209,140,.1) 100%)",
    display: "flex",
    flexDirection: "column",
    minHeight: "100vh",
  },
  content: {
    margin: theme.spacing(2, 0),
    [theme.breakpoints.up("md")]: {
      margin: theme.spacing(4, 0),
    },
  },
  headerImage: {
    position: "absolute",
    zIndex: -1,
    top: 0,
    background: `url(${Header})`,
    backgroundRepeat: "no-repeat",
    backgroundPosition: "top -500px center",
    backgroundSize: "2070px 1155px",
    width: "100%",
    height: 1155,
  },
  brandLink: {
    display: "inline-flex",
    alignItems: "center",
    "&:hover": {
      textDecoration: "none",
    },
  },
  iconButton: {
    [theme.breakpoints.up("md")]: {
      marginRight: theme.spacing(2.5),
    },
    [theme.breakpoints.down("sm")]: {
      marginRight: theme.spacing(2.5),
    },
    [theme.breakpoints.down("xs")]: {
      marginRight: theme.spacing(1),
    },
  },
  betaBanner: {
    backgroundColor: "rgba(0,0,0,0.75)",
    padding: theme.spacing(1, 0),
  },
  wormholeIcon: {
    height: 68,
    "&:hover": {
      filter: "contrast(1)",
    },
    verticalAlign: "middle",
    marginRight: theme.spacing(1),
    display: "inline-block",
  },
}));

function App() {
  const classes = useStyles();
  const isBeta = useBetaContext();
  const { push } = useHistory();
  const { pathname } = useLocation();
  const handleTabChange = useCallback(
    (event, value) => {
      push(value);
    },
    [push]
  );
  return (
    <div className={classes.bg}>
      <AppBar
        position="static"
        color="inherit"
        className={classes.appBar}
        elevation={0}
      >
        <Toolbar>
          <Link
            component={RouterLink}
            to="/transfer"
            className={classes.brandLink}
          >
            <img src={Portal} alt="Portal" className={classes.wormholeIcon} />
          </Link>
          <div className={classes.spacer} />
          <Hidden implementation="css" xsDown>
            <div style={{ display: "flex", alignItems: "center" }}>
              <Link
                component={NavLink}
                to="/transfer"
                color="inherit"
                className={classes.link}
              >
                Bridge
              </Link>
              <Link
                href="https://docs.wormholenetwork.com/wormhole/faqs"
                target="_blank"
                rel="noopener noreferrer"
                color="inherit"
                className={classes.link}
              >
                FAQ
              </Link>
              <Link
                component={NavLink}
                to="/stats"
                size="small"
                color="inherit"
                className={classes.link}
              >
                Stats
              </Link>
              <Link
                href="https://wormholenetwork.com/"
                target="_blank"
                rel="noopener noreferrer"
                color="inherit"
                className={classes.link}
              >
                Wormhole
              </Link>
            </div>
          </Hidden>
          <Hidden implementation="css" smUp>
            <Tooltip title="View the FAQ">
              <IconButton
                href="https://docs.wormholenetwork.com/wormhole/faqs"
                target="_blank"
                rel="noopener noreferrer"
                size="small"
                className={classes.link}
              >
                <HelpOutline />
              </IconButton>
            </Tooltip>
          </Hidden>
        </Toolbar>
      </AppBar>
      {CLUSTER === "mainnet" ? null : (
        <AppBar position="static" className={classes.betaBanner} elevation={0}>
          <Typography style={{ textAlign: "center" }}>
            Caution! You are using the {CLUSTER} build of this app.
          </Typography>
        </AppBar>
      )}
      {isBeta ? (
        <AppBar position="static" className={classes.betaBanner} elevation={0}>
          <Typography style={{ textAlign: "center" }}>
            Caution! You have enabled the beta. Enter the secret code again to
            disable.
          </Typography>
        </AppBar>
      ) : null}
      <div className={classes.content}>
        <div className={classes.headerImage} />
        {["/transfer", "/nft", "/redeem"].includes(pathname) ? (
          <Container maxWidth="md" style={{ paddingBottom: 24 }}>
            <HeaderText
              white
              subtitle={
                <>
                  <Typography>
                    Portal is a bridge that offers unlimited transfers across
                    chains for tokens and NFTs wrapped by Wormhole.
                  </Typography>
                  <Typography>
                    Unlike many other bridges, you avoid double wrapping and
                    never have to retrace your steps.
                  </Typography>
                </>
              }
            >
              Portal Token Bridge
            </HeaderText>
            <Tabs
              value={pathname}
              variant="fullWidth"
              onChange={handleTabChange}
              indicatorColor="primary"
            >
              <Tab label="Tokens" value="/transfer" />
              <Tab label="NFTs" value="/nft" />
              <Tab label="Redeem" value="/redeem" to="/redeem" />
            </Tabs>
          </Container>
        ) : null}
        <Switch>
          <Route exact path="/transfer">
            <Transfer />
          </Route>
          <Route exact path="/nft">
            <NFT />
          </Route>
          <Route exact path="/redeem">
            <Recovery />
          </Route>
          <Route exact path="/nft-origin-verifier">
            <NFTOriginVerifier />
          </Route>
          <Route exact path="/token-origin-verifier">
            <TokenOriginVerifier />
          </Route>
          <Route exact path="/register">
            <Attest />
          </Route>
          <Route exact path="/migrate/Solana/:legacyAsset/:fromTokenAccount">
            <Migration chainId={CHAIN_ID_SOLANA} />
          </Route>
          <Route exact path="/migrate/Ethereum/:legacyAsset/">
            <Migration chainId={CHAIN_ID_ETH} />
          </Route>
          <Route exact path="/migrate/BinanceSmartChain/:legacyAsset/">
            <Migration chainId={CHAIN_ID_BSC} />
          </Route>
          <Route exact path="/migrate/Ethereum/">
            <EvmQuickMigrate chainId={CHAIN_ID_ETH} />
          </Route>
          <Route exact path="/migrate/BinanceSmartChain/">
            <EvmQuickMigrate chainId={CHAIN_ID_BSC} />
          </Route>
          <Route exact path="/migrate/Solana/">
            <SolanaQuickMigrate />
          </Route>
          <Route exact path="/stats">
            <Stats />
          </Route>
          <Route exact path="/withdraw-tokens-terra">
            <WithdrawTokensTerra />
          </Route>
          <Route>
            <Redirect to="/transfer" />
          </Route>
        </Switch>
      </div>
      <div className={classes.spacer} />
      <Footer />
    </div>
  );
}

export default App;
