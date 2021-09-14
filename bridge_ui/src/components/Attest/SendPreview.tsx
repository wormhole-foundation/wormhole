import { makeStyles, Typography } from "@material-ui/core";
import { useSelector } from "react-redux";
import {
  selectAttestSourceChain,
  selectAttestAttestTx,
} from "../../store/selectors";
import ShowTx from "../ShowTx";

const useStyles = makeStyles((theme) => ({
  description: {
    textAlign: "center",
  },
  tx: {
    marginTop: theme.spacing(1),
    textAlign: "center",
  },
  viewButton: {
    marginTop: theme.spacing(1),
  },
}));

export default function SendPreview() {
  const classes = useStyles();
  const sourceChain = useSelector(selectAttestSourceChain);
  const attestTx = useSelector(selectAttestAttestTx);

  const explainerString = "The token has been attested!";

  return (
    <>
      <Typography
        component="div"
        variant="subtitle2"
        className={classes.description}
      >
        {explainerString}
      </Typography>
      {attestTx ? <ShowTx chainId={sourceChain} tx={attestTx} /> : null}
    </>
  );
}
