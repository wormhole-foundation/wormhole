import {
  Container,
  Divider,
  makeStyles,
  Paper,
  Typography,
} from "@material-ui/core";
import { PublicKey } from "@solana/web3.js";
import { RouteComponentProps } from "react-router-dom";
import {
  ETH_MIGRATION_ASSET_MAP,
  MIGRATION_ASSET_MAP,
} from "../../utils/consts";
import SolanaWorkflow from "./SolanaWorkflow";
import { withRouter } from "react-router";
import { COLORS } from "../../muiTheme";
import {
  ChainId,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
} from "@certusone/wormhole-sdk";
import EthereumWorkflow from "./EthereumWorkflow";

const useStyles = makeStyles(() => ({
  mainPaper: {
    backgroundColor: COLORS.nearBlackWithMinorTransparency,
    textAlign: "center",
    padding: "2rem",
    "& > h, p ": {
      margin: ".5rem",
    },
  },
  divider: {
    margin: "2rem 0rem 2rem 0rem",
  },
  spacer: {
    height: "2rem",
  },
}));

interface RouteParams {
  legacyAsset: string;
  fromTokenAccount: string;
}

interface Migration extends RouteComponentProps<RouteParams> {
  chainId: ChainId;
}

const SolanaRoot: React.FC<Migration> = (props) => {
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

  let content = null;

  if (!fromMint || !toMint) {
    content = (
      <Typography style={{ textAlign: "center" }}>
        This asset is not eligible for migration.
      </Typography>
    );
  } else if (!fromTokenAcct) {
    content = (
      <Typography style={{ textAlign: "center" }}>
        Invalid token account.
      </Typography>
    );
  } else {
    content = (
      <SolanaWorkflow
        fromMint={fromMint}
        toMint={toMint}
        fromTokenAccount={fromTokenAcct}
      />
    );
  }

  return content;
};

const EthereumRoot: React.FC<Migration> = (props) => {
  const legacyAsset: string = props.match.params.legacyAsset;
  const targetPool = ETH_MIGRATION_ASSET_MAP.get(legacyAsset);

  let content = null;
  if (!legacyAsset || !targetPool) {
    content = (
      <Typography style={{ textAlign: "center" }}>
        This asset is not eligible for migration.
      </Typography>
    );
  } else {
    content = <EthereumWorkflow migratorAddress={targetPool} />;
  }

  return content;
};

const MigrationRoot: React.FC<Migration> = (props) => {
  const classes = useStyles();
  let content = null;

  if (props.chainId === CHAIN_ID_SOLANA) {
    content = <SolanaRoot {...props} />;
  } else if (props.chainId === CHAIN_ID_ETH) {
    content = <EthereumRoot {...props} />;
  }

  return (
    <Container maxWidth="md">
      <Paper className={classes.mainPaper}>
        <Typography variant="h5">Migrate Assets</Typography>
        <Typography variant="subtitle2">
          Convert assets from other bridges to Wormhole V2 tokens
        </Typography>
        <Divider className={classes.divider} />
        {content}
      </Paper>
    </Container>
  );
};

export default withRouter(MigrationRoot);
