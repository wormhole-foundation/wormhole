import { AppBar, makeStyles, Toolbar } from "@material-ui/core";
import Attest from "./components/Attest";
import Transfer from "./components/Transfer";
import wormholeLogo from "./icons/wormhole.svg";

const useStyles = makeStyles((theme) => ({
  appBar: {
    borderBottom: `.5px solid ${theme.palette.divider}`,
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
    color: theme.palette.text.primary,
    marginLeft: theme.spacing(6),
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
          <img src={wormholeLogo} alt="Wormhole Logo" />
          <div className={classes.spacer} />
        </Toolbar>
      </AppBar>
      <div className={classes.content}>
        <Attest />
        <Transfer />
      </div>
    </>
  );
}

export default App;
