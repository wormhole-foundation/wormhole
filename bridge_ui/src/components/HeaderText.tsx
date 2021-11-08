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

export default function HeaderText({
  children,
  white,
  small,
}: {
  children: ReactChild;
  white?: boolean;
  small?: boolean;
}) {
  const classes = useStyles();
  return (
    <div className={classes.centeredContainer}>
      <Typography
        variant={small ? "h2" : "h1"}
        component="h1"
        className={clsx(classes.header, { [classes.linearGradient]: !white })}
      >
        {children}
      </Typography>
    </div>
  );
}
