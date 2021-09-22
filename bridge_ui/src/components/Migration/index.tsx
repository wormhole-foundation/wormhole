import { Typography } from "@material-ui/core";
import { PublicKey } from "@solana/web3.js";
import { RouteComponentProps } from "react-router-dom";
import { MIGRATION_ASSET_MAP } from "../../utils/consts";
import Workflow from "./Workflow";
import { withRouter } from "react-router";

interface RouteParams {
  legacyAsset: string;
  fromTokenAccount: string;
}

interface Migration extends RouteComponentProps<RouteParams> {}

const MigrationRoot: React.FC<Migration> = (props) => {
  const legacyAsset: string = props.match.params.legacyAsset;
  const fromTokenAccount: string = props.match.params.fromTokenAccount;
  const targetAsset: string | undefined = MIGRATION_ASSET_MAP.get(legacyAsset);

  let fromMint: string | undefined = "";
  let toMint: string | undefined = "";
  let fromTokenAcct: string | undefined = "";
  try {
    fromMint = legacyAsset && new PublicKey(legacyAsset).toString();
    toMint = targetAsset && new PublicKey(targetAsset).toString();
    fromTokenAcct =
      fromTokenAccount && new PublicKey(fromTokenAccount).toString();
  } catch (e) {}

  if (!fromMint || !toMint) {
    return (
      <Typography style={{ textAlign: "center" }}>
        This asset is not eligible for migration.
      </Typography>
    );
  } else if (!fromTokenAcct) {
    return (
      <Typography style={{ textAlign: "center" }}>
        Invalid token account.
      </Typography>
    );
  } else {
    return (
      <Workflow
        fromMint={fromMint}
        toMint={toMint}
        fromTokenAccount={fromTokenAcct}
      />
    );
  }
};

export default withRouter(MigrationRoot);
