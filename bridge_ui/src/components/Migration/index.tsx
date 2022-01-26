import {
  ChainId,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
} from "@certusone/wormhole-sdk";
import { getAddress } from "@ethersproject/address";
import { Container, makeStyles, Paper, Typography } from "@material-ui/core";
import { PublicKey } from "@solana/web3.js";
import { withRouter } from "react-router";
import { RouteComponentProps } from "react-router-dom";
import { COLORS } from "../../muiTheme";
import { getMigrationAssetMap, MIGRATION_ASSET_MAP } from "../../utils/consts";
import HeaderText from "../HeaderText";
import EvmWorkflow from "./EvmWorkflow";
import SolanaWorkflow from "./SolanaWorkflow";

const useStyles = makeStyles(() => ({
  mainPaper: {
    backgroundColor: COLORS.whiteWithTransparency,
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
  const assetMap = getMigrationAssetMap(props.chainId);
  const targetPool = assetMap.get(getAddress(legacyAsset));

  let content = null;
  if (!legacyAsset || !targetPool) {
    content = (
      <Typography style={{ textAlign: "center" }}>
        This asset is not eligible for migration.
      </Typography>
    );
  } else {
    content = (
      <EvmWorkflow migratorAddress={targetPool} chainId={props.chainId} />
    );
  }

  return content;
};

const MigrationRoot: React.FC<Migration> = (props) => {
  const classes = useStyles();
  let content = null;

  if (props.chainId === CHAIN_ID_SOLANA) {
    content = <SolanaRoot {...props} />;
  } else if (props.chainId === CHAIN_ID_ETH || props.chainId === CHAIN_ID_BSC) {
    content = <EthereumRoot {...props} />;
  }

  return (
    <Container maxWidth="md">
      <HeaderText
        white
        subtitle="Convert assets from other bridges to Wormhole V2 tokens"
      >
        Migrate Assets
      </HeaderText>
      <Paper className={classes.mainPaper}>{content}</Paper>
    </Container>
  );
};

export default withRouter(MigrationRoot);
