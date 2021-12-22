import {
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
} from "@certusone/wormhole-sdk";
import {
  AppBar,
  Button,
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
import { BarChart, HelpOutline, Send } from "@material-ui/icons";
import clsx from "clsx";
import { useCallback } from "react";
import { useHistory, useLocation, useRouteMatch } from "react-router";
import {
  Link as RouterLink,
  NavLink,
  Redirect,
  Route,
  Switch,
} from "react-router-dom";
import Attest from "./components/Attest";
import Footer from "./components/Footer";
import Home from "./components/Home";
import Migration from "./components/Migration";
import EvmQuickMigrate from "./components/Migration/EvmQuickMigrate";
import NFT from "./components/NFT";
import NFTOriginVerifier from "./components/NFTOriginVerifier";
import Recovery from "./components/Recovery";
import Transfer from "./components/Transfer";
import { useBetaContext } from "./contexts/BetaContext";
import { COLORS } from "./muiTheme";
import { CLUSTER } from "./utils/consts";
import Stats from "./components/Stats";
import TokenOriginVerifier from "./components/TokenOriginVerifier";
import SolanaQuickMigrate from "./components/Migration/SolanaQuickMigrate";
import Wormhole from "./icons/wormhole-network.svg";
import WithdrawTokensTerra from "./components/WithdrawTokensTerra";

const useStyles = makeStyles((theme) => ({
  appBar: {
    background: COLORS.nearBlackWithMinorTransparency,
    "& > .MuiToolbar-root": {
      margin: "auto",
      width: "100%",
      maxWidth: 1100,
    },
  },
  spacer: {
    flex: 1,
    width: "100vw",
  },
  link: {
    ...theme.typography.body1,
    color: theme.palette.text.primary,
    marginLeft: theme.spacing(6),
    [theme.breakpoints.down("sm")]: {
      marginLeft: theme.spacing(2.5),
    },
    [theme.breakpoints.down("xs")]: {
      marginLeft: theme.spacing(1),
    },
    "&.active": {
      color: theme.palette.primary.light,
    },
  },
  bg: {
    background:
      "linear-gradient(160deg, rgba(69,74,117,.1) 0%, rgba(138,146,178,.1) 33%, rgba(69,74,117,.1) 66%, rgba(98,104,143,.1) 100%), linear-gradient(45deg, rgba(153,69,255,.1) 0%, rgba(121,98,231,.1) 20%, rgba(0,209,140,.1) 100%)",
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
  brandLink: {
    display: "inline-flex",
    alignItems: "center",
    "&:hover": {
      textDecoration: "none",
    },
  },
  brandText: {
    ...theme.typography.h5,
    [theme.breakpoints.down("xs")]: {
      fontSize: 22,
    },
    fontWeight: "500",
    background: `linear-gradient(160deg, rgba(255,255,255,1) 0%, rgba(255,255,255,0.5) 100%);`,
    WebkitBackgroundClip: "text",
    backgroundClip: "text",
    WebkitTextFillColor: "transparent",
    MozBackgroundClip: "text",
    MozTextFillColor: "transparent",
    letterSpacing: "3px",
    display: "inline-block",
    marginLeft: theme.spacing(0.5),
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
  gradientButton: {
    backgroundImage: `linear-gradient(45deg, ${COLORS.blue} 0%, ${COLORS.nearBlack}20 50%,  ${COLORS.blue}30 62%, ${COLORS.nearBlack}50  120%)`,
    transition: "0.75s",
    backgroundSize: "200% auto",
    boxShadow: "0 0 20px #222",
    "&:hover": {
      backgroundPosition:
        "right center" /* change the direction of the change here */,
    },
  },
  betaBanner: {
    background: `linear-gradient(to left, ${COLORS.blue}40, ${COLORS.green}40);`,
    padding: theme.spacing(1, 0),
  },
  wormholeIcon: {
    height: 32,
    filter: "contrast(0)",
    transition: "filter 0.5s",
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
  const isHomepage = useRouteMatch({ path: "/", exact: true });
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
      <AppBar position="static" color="inherit" className={classes.appBar}>
        <Toolbar>
          <Link component={RouterLink} to="/" className={classes.brandLink}>
            <img
              src={Wormhole}
              alt="Wormhole"
              className={classes.wormholeIcon}
            />
            <Typography className={clsx(classes.link, classes.brandText)}>
              wormhole
            </Typography>
          </Link>
          <div className={classes.spacer} />
          <Hidden implementation="css" xsDown>
            <div style={{ display: "flex", alignItems: "center" }}>
              {isHomepage ? (
                <>
                  <Tooltip title="View wormhole network stats">
                    <IconButton
                      component={NavLink}
                      to="/stats"
                      size="small"
                      className={clsx(classes.link, classes.iconButton)}
                    >
                      <BarChart />
                    </IconButton>
                  </Tooltip>
                  <Button
                    component={RouterLink}
                    to="/transfer"
                    variant="contained"
                    color="primary"
                    size="large"
                    className={classes.gradientButton}
                  >
                    Transfer Tokens
                  </Button>
                </>
              ) : (
                <Tooltip title="View the FAQ">
                  <Button
                    href="https://docs.wormholenetwork.com/wormhole/faqs"
                    target="_blank"
                    rel="noopener noreferrer"
                    variant="outlined"
                    endIcon={<HelpOutline />}
                  >
                    FAQ
                  </Button>
                </Tooltip>
              )}
            </div>
          </Hidden>
          <Hidden implementation="css" smUp>
            {isHomepage ? (
              <>
                <Tooltip title="View wormhole network stats">
                  <IconButton
                    component={NavLink}
                    to="/stats"
                    size="small"
                    className={classes.link + " " + classes.iconButton}
                  >
                    <BarChart />
                  </IconButton>
                </Tooltip>
                <Tooltip title="Transfer tokens to another blockchain">
                  <IconButton
                    component={NavLink}
                    to="/transfer"
                    size="small"
                    className={classes.link}
                  >
                    <Send />
                  </IconButton>
                </Tooltip>
              </>
            ) : (
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
            )}
          </Hidden>
        </Toolbar>
      </AppBar>
      {CLUSTER === "mainnet" ? null : (
        <AppBar position="static" className={classes.betaBanner}>
          <Typography style={{ textAlign: "center" }}>
            Caution! You are using the {CLUSTER} build of this app.
          </Typography>
        </AppBar>
      )}
      {isBeta ? (
        <AppBar position="static" className={classes.betaBanner}>
          <Typography style={{ textAlign: "center" }}>
            Caution! You have enabled the beta. Enter the secret code again to
            disable.
          </Typography>
        </AppBar>
      ) : null}
      <div className={classes.content}>
        {["/transfer", "/nft", "/redeem"].includes(pathname) ? (
          <Container maxWidth="md" style={{ paddingBottom: 24 }}>
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
          <Route exact path="/">
            <Home />
          </Route>
          <Route>
            <Redirect to="/" />
          </Route>
        </Switch>
      </div>
      <div className={classes.spacer} />
      <Footer />
    </div>
  );
}

export default App;
