import { makeStyles, Typography } from "@material-ui/core";
import clsx from "clsx";
import { ReactChild } from "react";
import { COLORS } from "../muiTheme";

const useStyles = makeStyles((theme) => ({
  centeredContainer: {
    textAlign: "center",
    width: "100%",
  },
  header: {
    marginTop: theme.spacing(2),
    marginBottom: theme.spacing(4),
    [theme.breakpoints.down("sm")]: {
      marginBottom: theme.spacing(4),
    },
  },
  linearGradient: {
    background: `linear-gradient(to left, ${COLORS.blue}, ${COLORS.green});`,
    WebkitBackgroundClip: "text",
    backgroundClip: "text",
    WebkitTextFillColor: "transparent",
    MozBackgroundClip: "text",
    MozTextFillColor: "transparent",
  },
}));

export default function HeaderText({ children }: { children: ReactChild }) {
  const classes = useStyles();
  return (
    <div className={classes.centeredContainer}>
      <Typography
        variant="h1"
        className={clsx(classes.header, classes.linearGradient)}
      >
        {children}
      </Typography>
    </div>
  );
}
