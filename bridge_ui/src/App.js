import {
  AppBar,
  Hidden,
  IconButton,
  Link,
  makeStyles,
  Toolbar,
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
          <img src={wormholeLogo} alt="Wormhole Logo" style={{ height: 45 }} />
          <div className={classes.spacer} />
          <Hidden implementation="css" xsDown>
            <div style={{ display: "flex", alignItems: "center" }}>
              <Link component={NavLink} to="/transfer" className={classes.link}>
                Transfer
              </Link>
              <Link component={NavLink} to="/attest" className={classes.link}>
                Attest
              </Link>
              <IconButton
                href="https://github.com/certusone/wormhole"
                target="_blank"
                size="small"
                className={classes.link}
              >
                <GitHub />
              </IconButton>
            </div>
          </Hidden>
          <Hidden implementation="css" smUp>
            <IconButton
              component={NavLink}
              to="/transfer"
              size="small"
              className={classes.link}
            >
              <Send />
            </IconButton>
            <IconButton
              component={NavLink}
              to="/attest"
              size="small"
              className={classes.link}
            >
              <Publish />
            </IconButton>
          </Hidden>
        </Toolbar>
      </AppBar>
      <div className={classes.content}>
        <Switch>
          <Route exact path="/transfer">
            <Transfer />
          </Route>
          <Route exact path="/attest">
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
