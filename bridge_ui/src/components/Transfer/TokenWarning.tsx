import {
  ChainId,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  isEVMChain,
} from "@certusone/wormhole-sdk";
import { Box, Link, makeStyles, Typography } from "@material-ui/core";
import { Alert } from "@material-ui/lab";
import {
  AVAILABLE_MARKETS_URL,
  CHAINS_BY_ID,
  MULTI_CHAIN_TOKENS,
} from "../../utils/consts";

const useStyles = makeStyles((theme) => ({
  container: {
    marginTop: theme.spacing(2),
    marginBottom: theme.spacing(2),
  },
  alert: {
    textAlign: "center",
  },
  line: {
    marginBottom: theme.spacing(2),
  },
}));

function WormholeWrappedWarning() {
  const classes = useStyles();
  return (
    <Alert severity="info" variant="outlined" className={classes.alert}>
      <Typography component="div" className={classes.line}>
        The tokens you will receive are{" "}
        <Box fontWeight={900} display="inline">
          Portal Wrapped Tokens
        </Box>{" "}
        and will need to be exchanged for native assets.
      </Typography>
      <Typography component="div">
        <Link
          href={AVAILABLE_MARKETS_URL}
          target="_blank"
          rel="noopener noreferrer"
        >
          Click here to see available markets for wrapped tokens.
        </Link>
      </Typography>
    </Alert>
  );
}

function MultichainWarning({
  symbol,
  targetChain,
}: {
  symbol: string;
  targetChain: ChainId;
}) {
  const classes = useStyles();
  return (
    <Alert severity="warning" variant="outlined" className={classes.alert}>
      <Typography
        variant="h6"
        className={classes.line}
      >{`You will not receive native ${symbol} on ${CHAINS_BY_ID[targetChain].name}`}</Typography>
      <Typography
        className={classes.line}
      >{`To receive native ${symbol}, you will have to perform a swap with the wrapped tokens once you are done bridging.`}</Typography>
      <Typography>
        <Link
          href={AVAILABLE_MARKETS_URL}
          target="_blank"
          rel="noopener noreferrer"
        >
          Click here to see available markets for wrapped tokens.
        </Link>
      </Typography>
    </Alert>
  );
}

function RewardsWarning() {
  const classes = useStyles();
  return (
    <Alert severity="warning" variant="outlined" className={classes.alert}>
      Lido stETH rewards can only be received on Ethereum. Use the value
      accruing wrapper token wstETH instead.
    </Alert>
  );
}

function LiquidityWarning() {
  const classes = useStyles();
  return (
    <Alert severity="info" variant="outlined" className={classes.alert}>
      <Typography component="div" className={classes.line}>
        The tokens you will receive are{" "}
        <Box fontWeight={900} display="inline">
          Portal Wrapped Tokens
        </Box>{" "}
        which currently have no liquid markets!
      </Typography>
      <Typography component="div">
        <Link
          href={AVAILABLE_MARKETS_URL}
          target="_blank"
          rel="noopener noreferrer"
        >
          Click here to see available markets for wrapped tokens.
        </Link>
      </Typography>
    </Alert>
  );
}

function shouldShowLiquidityWarning(
  sourceChain: ChainId,
  sourceAsset: string,
  targetChain: ChainId
) {
  if (sourceChain === CHAIN_ID_SOLANA && targetChain === CHAIN_ID_BSC) {
    return [
      "7i5KKsX2weiTkry7jA4ZwSuXGhs5eJBEjY8vVxR4pfRx", // GMT
      "AFbX8oGjGpmVFywbVouvhQSRmiW2aR1mohfahi4Y2AdB", // GST
    ].includes(sourceAsset);
  } else if (sourceChain === CHAIN_ID_BSC && targetChain === CHAIN_ID_SOLANA) {
    return [
      "0x3019bf2a2ef8040c242c9a4c5c4bd4c81678b2a1", // GMT
      "0x4a2c860cec6471b9f5f5a336eb4f38bb21683c98", // GST
      "0x570a5d26f7765ecb712c0924e4de545b89fd43df", // "sol"
    ].includes(sourceAsset);
  }
  return false;
}

export default function TokenWarning({
  sourceChain,
  sourceAsset,
  originChain,
  targetChain,
  targetAsset,
}: {
  sourceChain?: ChainId;
  sourceAsset?: string;
  originChain?: ChainId;
  targetChain?: ChainId;
  targetAsset?: string;
}) {
  if (
    !(originChain && targetChain && targetAsset && sourceChain && sourceAsset)
  ) {
    return null;
  }

  const searchableAddress = isEVMChain(sourceChain)
    ? sourceAsset.toLowerCase()
    : sourceAsset;
  const isWormholeWrapped = originChain !== targetChain;
  const multichainSymbol =
    MULTI_CHAIN_TOKENS[sourceChain]?.[searchableAddress] || undefined;
  const isMultiChain = !!multichainSymbol;
  const isRewardsToken =
    searchableAddress === "0xae7ab96520de3a18e5e111b5eaab095312d7fe84" &&
    sourceChain === CHAIN_ID_ETH;

  const showMultiChainWarning = isMultiChain && isWormholeWrapped;
  const showWrappedWarning = !isMultiChain && isWormholeWrapped; //Multichain warning is more important
  const showRewardsWarning = isRewardsToken;
  const showLiquidityWarning = shouldShowLiquidityWarning(
    sourceChain,
    searchableAddress,
    targetChain
  );

  return (
    <>
      {showMultiChainWarning ? (
        <MultichainWarning
          symbol={multichainSymbol || "tokens"}
          targetChain={targetChain}
        />
      ) : null}
      {showWrappedWarning ? <WormholeWrappedWarning /> : null}
      {showRewardsWarning ? <RewardsWarning /> : null}
      {showLiquidityWarning ? <LiquidityWarning /> : null}
    </>
  );
}
