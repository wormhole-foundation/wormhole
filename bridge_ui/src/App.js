import {
  AppBar,
  Hidden,
  IconButton,
  Link,
  makeStyles,
  Toolbar,
  Tooltip,
  Typography,
} from "@material-ui/core";
import { GitHub, Help, Publish, Send } from "@material-ui/icons";
import {
  Link as RouterLink,
  NavLink,
  Redirect,
  Route,
  Switch,
} from "react-router-dom";
import Attest from "./components/Attest";
import Home from "./components/Home";
import Migration from "./components/Migration";
import NFT from "./components/NFT";
import NFTOriginVerifier from "./components/NFTOriginVerifier";
import Transfer from "./components/Transfer";
import wormholeLogo from "./icons/wormhole.svg";
import { CLUSTER } from "./utils/consts";

const useStyles = makeStyles((theme) => ({
  appBar: {
    borderBottom: `1px solid ${theme.palette.divider}`,
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
  logo: {
    verticalAlign: "middle",
    height: 52,
    [theme.breakpoints.down("xs")]: {
      height: 42,
    },
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
      color: theme.palette.secondary.light,
    },
  },
  content: {
    [theme.breakpoints.up("sm")]: {
      margin: theme.spacing(2, 0),
    },
    [theme.breakpoints.up("md")]: {
      margin: theme.spacing(4, 0),
    },
  },
}));

function App() {
  const classes = useStyles();
  return (
    <>
      <AppBar position="static" color="inherit" className={classes.appBar}>
        <Toolbar>
          <RouterLink to="/">
            <img
              src={wormholeLogo}
              alt="Wormhole Logo"
              className={classes.logo}
            />
          </RouterLink>
          <div className={classes.spacer} />
          <Hidden implementation="css" xsDown>
            <div style={{ display: "flex", alignItems: "center" }}>
              <Tooltip title="Transfer NFTs to another blockchain">
                <Link component={NavLink} to="/nft" className={classes.link}>
                  NFTs
                </Link>
              </Tooltip>
              <Tooltip title="Transfer tokens to another blockchain">
                <Link
                  component={NavLink}
                  to="/transfer"
                  className={classes.link}
                >
                  Transfer
                </Link>
              </Tooltip>
              <Tooltip title="Register a new wrapped token">
                <Link
                  component={NavLink}
                  to="/register"
                  className={classes.link}
                >
                  Register
                </Link>
              </Tooltip>
              <Tooltip title="View the source code">
                <IconButton
                  href="https://github.com/certusone/wormhole"
                  target="_blank"
                  size="small"
                  className={classes.link}
                >
                  <GitHub />
                </IconButton>
              </Tooltip>
              <Tooltip title="View the FAQ">
                <IconButton
                  href="https://docs.wormholenetwork.com/wormhole/faqs"
                  target="_blank"
                  size="small"
                  className={classes.link}
                >
                  <Help />
                </IconButton>
              </Tooltip>
            </div>
          </Hidden>
          <Hidden implementation="css" smUp>
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
            <Tooltip title="Register a new wrapped token">
              <IconButton
                component={NavLink}
                to="/register"
                size="small"
                className={classes.link}
              >
                <Publish />
              </IconButton>
            </Tooltip>
            <Tooltip title="View the FAQ">
              <IconButton
                href="https://docs.wormholenetwork.com/wormhole/faqs"
                target="_blank"
                size="small"
                className={classes.link}
              >
                <Help />
              </IconButton>
            </Tooltip>
          </Hidden>
        </Toolbar>
      </AppBar>
      {CLUSTER === "mainnet" ? null : (
        <AppBar position="static" color="secondary">
          <Typography style={{ textAlign: "center" }}>
            Caution! You are using the {CLUSTER} build of this app.
          </Typography>
        </AppBar>
      )}
      <div className={classes.content}>
        <Switch>
          <Route exact path="/nft">
            <NFT />
          </Route>
          <Route exact path="/nft-origin-verifier">
            <NFTOriginVerifier />
          </Route>
          <Route exact path="/transfer">
            <Transfer />
          </Route>
          <Route exact path="/register">
            <Attest />
          </Route>
          <Route exact path="/migrate/:legacyAsset/:fromTokenAccount">
            <Migration />
          </Route>
          <Route exact path="/">
            <Home />
          </Route>
          <Route>
            <Redirect to="/" />
          </Route>
        </Switch>
      </div>
    </>
  );
}

export default App;
