import {
  ChainId,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
} from "@certusone/wormhole-sdk";
import { Button, makeStyles, Tooltip, Typography } from "@material-ui/core";
import { withStyles } from "@material-ui/styles";
import { useSnackbar } from "notistack";
import { useCallback } from "react";
import { ParsedTokenAccount } from "../store/transferSlice";
import { CLUSTER } from "../utils/consts";
import { shortenAddress } from "../utils/solana";
import { FileCopy, OpenInNew } from "@material-ui/icons";

const useStyles = makeStyles((theme) => ({
  mainTypog: {
    display: "inline-block",
    marginLeft: theme.spacing(1),
    marginRight: theme.spacing(1),
    textDecoration: "underline",
    textUnderlineOffset: "2px",
  },
  buttons: {
    marginLeft: ".5rem",
    marginRight: ".5rem",
  },
}));

function pushToClipboard(content: any) {
  if (!navigator.clipboard) {
    // Clipboard API not available
    return;
  }
  return navigator.clipboard.writeText(content);
}

const tooltipStyles = {
  tooltip: {
    minWidth: "max-content",
    textAlign: "center",
    "& > *": {
      margin: ".25rem",
    },
  },
};

// @ts-ignore
const StyledTooltip = withStyles(tooltipStyles)(Tooltip);

export default function SmartAddress({
  chainId,
  parsedTokenAccount,
  address,
  symbol,
  tokenName,
  variant,
}: {
  chainId: ChainId;
  parsedTokenAccount?: ParsedTokenAccount;
  address?: string;
  logo?: string;
  tokenName?: string;
  symbol?: string;
  variant?: any;
}) {
  const classes = useStyles();
  const useableAddress = parsedTokenAccount?.mintKey || address || "";
  const useableSymbol = parsedTokenAccount?.symbol || symbol || "";
  const isNative = parsedTokenAccount?.isNativeAsset || false;
  const addressShort = shortenAddress(useableAddress) || "";
  const { enqueueSnackbar } = useSnackbar();

  const useableName = isNative
    ? "Native Currency"
    : parsedTokenAccount?.name
    ? parsedTokenAccount.name
    : tokenName
    ? tokenName
    : "";
  //TODO terra
  const explorerAddress = isNative
    ? null
    : chainId === CHAIN_ID_ETH
    ? `https://${
        CLUSTER === "testnet" ? "goerli." : ""
      }etherscan.io/address/${useableAddress}`
    : chainId === CHAIN_ID_SOLANA
    ? `https://explorer.solana.com/address/${useableAddress}${
        CLUSTER === "testnet"
          ? "?cluster=testnet"
          : CLUSTER === "devnet"
          ? "?cluster=custom&customUrl=http%3A%2F%2Flocalhost%3A8899"
          : ""
      }`
    : undefined;
  const explorerName = chainId === CHAIN_ID_ETH ? "Etherscan" : "Explorer";

  const copyToClipboard = useCallback(() => {
    pushToClipboard(useableAddress)?.then(() => {
      enqueueSnackbar("Copied address to clipboard.", { variant: "success" });
    });
  }, [useableAddress, enqueueSnackbar]);

  const explorerButton = !explorerAddress ? null : (
    <Button
      size="small"
      variant="outlined"
      endIcon={<OpenInNew />}
      className={classes.buttons}
      href={explorerAddress}
      target="_blank"
    >
      {"View on " + explorerName}
    </Button>
  );
  //TODO add icon here
  const copyButton = isNative ? null : (
    <Button
      size="small"
      variant="outlined"
      endIcon={<FileCopy />}
      onClick={copyToClipboard}
      className={classes.buttons}
    >
      Copy
    </Button>
  );

  const tooltipContent = (
    <>
      {useableName && <Typography>{useableName}</Typography>}
      {useableSymbol && !isNative && (
        <Typography noWrap variant="body2">
          {addressShort}
        </Typography>
      )}
      <div>
        {explorerButton}
        {copyButton}
      </div>
    </>
  );

  return (
    <StyledTooltip
      title={tooltipContent}
      interactive={true}
      className={classes.mainTypog}
    >
      <Typography
        variant={variant || "body1"}
        className={classes.mainTypog}
        component="div"
      >
        {useableSymbol || addressShort}
      </Typography>
    </StyledTooltip>
  );
}
