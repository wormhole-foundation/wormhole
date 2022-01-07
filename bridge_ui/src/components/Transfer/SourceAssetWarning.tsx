import { ChainId, CHAIN_ID_POLYGON, isEVMChain } from "@certusone/wormhole-sdk";
import { makeStyles, Typography } from "@material-ui/core";
import { Alert } from "@material-ui/lab";
import { POLYGON_TERRA_WRAPPED_TOKENS } from "../../utils/consts";

const useStyles = makeStyles((theme) => ({
  container: {
    marginTop: theme.spacing(2),
    marginBottom: theme.spacing(2),
  },
  alert: {
    marginTop: theme.spacing(1),
    marginBottom: theme.spacing(1),
  },
}));

function PolygonTerraWrappedWarning() {
  const classes = useStyles();
  return (
    <Alert severity="warning" variant="outlined" className={classes.alert}>
      <Typography variant="body1">
        This is a Shuttle-wrapped asset from Polygon! Transferring it will
        result in a double wrapped (Wormhole-wrapped Shuttle-wrapped) asset,
        which has no liquid markets.
      </Typography>
    </Alert>
  );
}

export default function SoureAssetWarning({
  sourceChain,
  sourceAsset,
}: {
  sourceChain?: ChainId;
  sourceAsset?: string;
  originChain?: ChainId;
  targetChain?: ChainId;
  targetAsset?: string;
}) {
  if (!(sourceChain && sourceAsset)) {
    return null;
  }

  const searchableAddress = isEVMChain(sourceChain)
    ? sourceAsset.toLowerCase()
    : sourceAsset;
  const showPolygonTerraWrappedWarning =
    sourceChain === CHAIN_ID_POLYGON &&
    POLYGON_TERRA_WRAPPED_TOKENS.includes(searchableAddress);

  return (
    <>
      {showPolygonTerraWrappedWarning ? <PolygonTerraWrappedWarning /> : null}
    </>
  );
}
