import { makeStyles, Typography } from "@material-ui/core";

const useStyles = makeStyles((theme) => ({
  description: {
    textAlign: "center",
  },
}));

export default function RedeemPreview() {
  const classes = useStyles();

  const explainerString =
    "Success! The redeem transaction was submitted. The tokens will become available once the transaction confirms.";

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
