import { makeStyles, Typography } from "@material-ui/core";
import { ReactChild } from "react";

const useStyles = makeStyles((theme) => ({
  description: {
    marginBottom: theme.spacing(4),
  },
}));

export default function StepDescription({
  children,
}: {
  children: ReactChild;
}) {
  const classes = useStyles();
  return (
    <Typography component="div" variant="body2" className={classes.description}>
      {children}
    </Typography>
  );
}
