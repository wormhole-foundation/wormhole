import { makeStyles } from "@material-ui/core";
import hole from "../images/hole.svg";

const useStyles = makeStyles((theme) => ({
  root: {
    position: "fixed",
    top: 0,
    right: 0,
    bottom: 0,
    left: 0,
    background: `radial-gradient(100% 100% at 100% 125%,${theme.palette.secondary.dark} 0,rgba(255,255,255,0) 100%)`,
    zIndex: -1,
  },
  hole: {
    position: "fixed",
    bottom: 0,
    right: 0,
    opacity: 0.3,
    filter: "blur(1px)",
    zIndex: -1,
  },
}));

const RadialGradient = () => {
  const classes = useStyles();
  return (
    <>
      <img src={hole} alt="" className={classes.hole} />
      <div className={classes.root} />
    </>
  );
};

export default RadialGradient;
