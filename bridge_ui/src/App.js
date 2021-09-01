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
import { GitHub, Publish, Send } from "@material-ui/icons";
import { NavLink, Redirect, Route, Switch } from "react-router-dom";
import Attest from "./components/Attest";
import Transfer from "./components/Transfer";
import wormholeLogo from "./icons/wormhole.svg";

const useStyles = makeStyles((theme) => ({
  appBar: {
    borderBottom: `1px solid ${theme.palette.divider}`,
    "& > .MuiToolbar-root": {
      margin: "auto",
      height: 69,
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
      marginLeft: theme.spacing(4),
    },
    [theme.breakpoints.down("xs")]: {
      marginLeft: theme.spacing(2),
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
          <img src={wormholeLogo} alt="Wormhole Logo" style={{ height: 52 }} />
          <div className={classes.spacer} />
          <Hidden implementation="css" xsDown>
            <div style={{ display: "flex", alignItems: "center" }}>
              <Tooltip title="Coming Soon">
                <Typography
                  className={classes.link}
                  style={{ color: "#ffffff80", cursor: "default" }}
                >
                  NFTs
                </Typography>
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
          </Hidden>
        </Toolbar>
      </AppBar>
      <div className={classes.content}>
        <Switch>
          <Route exact path="/transfer">
            <Transfer />
          </Route>
          <Route exact path="/register">
            <Attest />
          </Route>
          <Route>
            <Redirect to="/transfer" />
          </Route>
        </Switch>
      </div>
    </>
  );
}

export default App;
