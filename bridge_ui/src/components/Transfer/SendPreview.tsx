import { makeStyles, Typography } from "@material-ui/core";

const useStyles = makeStyles((theme) => ({
  description: {
    textAlign: "center",
  },
}));

export default function SendPreview() {
  const classes = useStyles();

  const explainerString = "The tokens have been sent!";

  return (
    <Typography
      component="div"
      variant="subtitle2"
      className={classes.description}
    >
      {explainerString}
    </Typography>
  );
}
