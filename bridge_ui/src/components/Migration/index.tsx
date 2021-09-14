import { Typography } from "@material-ui/core";
import { PublicKey } from "@solana/web3.js";
import { RouteComponentProps } from "react-router-dom";
import { MIGRATION_ASSET_MAP } from "../../utils/consts";
import Workflow from "./Workflow";
import { withRouter } from "react-router";

interface RouteParams {
  legacyAsset: string;
}

interface Migration extends RouteComponentProps<RouteParams> {}

const MigrationRoot: React.FC<Migration> = (props) => {
  const legacyAsset: string = props.match.params.legacyAsset;
  const targetAsset: string | undefined = MIGRATION_ASSET_MAP.get(legacyAsset);

  let fromMint: string | undefined = "";
  let toMint: string | undefined = "";
  try {
    fromMint = legacyAsset && new PublicKey(legacyAsset).toString();
    toMint = targetAsset && new PublicKey(targetAsset).toString();
  } catch (e) {}

  if (fromMint && toMint) {
    return <Workflow fromMint={fromMint} toMint={toMint} />;
  } else {
    return <Typography>This asset is not eligible for migration.</Typography>;
  }
};

export default withRouter(MigrationRoot);
