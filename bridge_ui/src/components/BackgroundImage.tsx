import { makeStyles } from "@material-ui/core";
import { useRouteMatch } from "react-router";
import holev2 from "../images/holev2.svg";

const useStyles = makeStyles((theme) => ({
  holeOuterContainer: {
    maxWidth: "100%",
    width: "100%",
    position: "relative",
  },
  holeInnerContainer: {
    position: "absolute",
    zIndex: -1,
    left: "50%",
    transform: "translate(-50%, 0)",
    width: "100%",
    maxWidth: "100%",
    overflow: "hidden",
    display: "flex",
    justifyContent: "center",
  },
  holeImage: {
    width: "max(1200px, 100vw)",
    maxWidth: "1600px",
  },
  blurred: {
    filter: "blur(2px)",
    opacity: ".9",
  },
}));

const BackgroundImage = () => {
  const classes = useStyles();
  const isHomepage = useRouteMatch({ path: "/", exact: true });

  return (
    <div className={classes.holeOuterContainer}>
      <div className={classes.holeInnerContainer}>
        <img
          src={holev2}
          alt=""
          className={
            classes.holeImage + (isHomepage ? "" : " " + classes.blurred)
          }
        />
      </div>
    </div>
  );
};

export default BackgroundImage;
