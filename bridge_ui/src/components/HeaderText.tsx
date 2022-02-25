import { makeStyles, Typography } from "@material-ui/core";
import clsx from "clsx";
import { ReactChild } from "react";
import { COLORS } from "../muiTheme";

const useStyles = makeStyles((theme) => ({
  centeredContainer: {
    marginTop: theme.spacing(14),
    marginBottom: theme.spacing(26),
    minHeight: 208,
    textAlign: "center",
    width: "100%",
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
  subtitle,
}: {
  children: ReactChild;
  white?: boolean;
  small?: boolean;
  subtitle?: ReactChild;
}) {
  const classes = useStyles();
  return (
    <div className={classes.centeredContainer}>
      <Typography
        variant={small ? "h2" : "h1"}
        component="h1"
        className={clsx({ [classes.linearGradient]: !white })}
        gutterBottom={!!subtitle}
      >
        {children}
      </Typography>
      {subtitle ? <Typography component="div">{subtitle}</Typography> : null}
    </div>
  );
}
