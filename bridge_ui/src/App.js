import { AppBar, Link, makeStyles, Toolbar } from "@material-ui/core";
import Transfer from "./components/Transfer";
import wormholeLogo from "./icons/wormhole.svg";

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
}));

function App() {
  const classes = useStyles();
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
        <Transfer />
      </div>
    </>
  );
}

export default App;
