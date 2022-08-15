import { Link, makeStyles } from "@material-ui/core";
import { Alert } from "@material-ui/lab";
import { useSelector } from "react-redux";
import { selectTransferSourceChain } from "../../store/selectors";
import { CHAINS_BY_ID } from "../../utils/consts";

const useStyles = makeStyles((theme) => ({
  alert: {
    marginTop: theme.spacing(1),
    marginBottom: theme.spacing(1),
  },
}));

const PendingVAAWarning = () => {
  const classes = useStyles();
  const sourceChain = useSelector(selectTransferSourceChain);
  const chainName = CHAINS_BY_ID[sourceChain]?.name || "unknown";
  const message = `The daily notional value limit for transfers on ${chainName} has been exceeded. As
      a result, the VAA for this transfer is pending. If you have any questions,
      please open a support ticket on `;
  return (
    <Alert variant="outlined" severity="warning" className={classes.alert}>
      {message}
      <Link
        href="https://discord.gg/wormholecrypto"
        target="_blank"
        rel="noopener noreferrer"
      >
        https://discord.gg/wormholecrypto
      </Link>
      {"."}
    </Alert>
  );
};

export default PendingVAAWarning;
