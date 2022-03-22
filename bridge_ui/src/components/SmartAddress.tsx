import {
  ChainId,
  CHAIN_ID_AURORA,
  CHAIN_ID_AVAX,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_ETHEREUM_ROPSTEN,
  CHAIN_ID_FANTOM,
  CHAIN_ID_OASIS,
  CHAIN_ID_POLYGON,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  isNativeDenom,
} from "@certusone/wormhole-sdk";
import { Button, makeStyles, Tooltip, Typography } from "@material-ui/core";
import { FileCopy, OpenInNew } from "@material-ui/icons";
import { withStyles } from "@material-ui/styles";
import clsx from "clsx";
import { ReactChild } from "react";
import useCopyToClipboard from "../hooks/useCopyToClipboard";
import { ParsedTokenAccount } from "../store/transferSlice";
import { CLUSTER, getExplorerName } from "../utils/consts";
import { shortenAddress } from "../utils/solana";
import { formatNativeDenom } from "../utils/terra";

const useStyles = makeStyles((theme) => ({
  mainTypog: {
    display: "inline-block",
    marginLeft: theme.spacing(1),
    marginRight: theme.spacing(1),
    textDecoration: "underline",
    textUnderlineOffset: "2px",
  },
  noGutter: {
    marginLeft: 0,
    marginRight: 0,
  },
  noUnderline: {
    textDecoration: "none",
  },
  buttons: {
    marginLeft: ".5rem",
    marginRight: ".5rem",
  },
}));

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
  noGutter,
  noUnderline,
  extraContent,
}: {
  chainId: ChainId;
  parsedTokenAccount?: ParsedTokenAccount;
  address?: string;
  logo?: string;
  tokenName?: string;
  symbol?: string;
  variant?: any;
  noGutter?: boolean;
  noUnderline?: boolean;
  extraContent?: ReactChild;
}) {
  const classes = useStyles();
  const isNativeTerra = chainId === CHAIN_ID_TERRA && isNativeDenom(address);
  const useableAddress = parsedTokenAccount?.mintKey || address || "";
  const useableSymbol = isNativeTerra
    ? formatNativeDenom(address)
    : parsedTokenAccount?.symbol || symbol || "";
  // const useableLogo = logo || isNativeTerra ? getNativeTerraIcon(useableSymbol) : null
  const isNative = parsedTokenAccount?.isNativeAsset || isNativeTerra || false;
  const addressShort = shortenAddress(useableAddress) || "";

  const useableName = isNative
    ? "Native Currency"
    : parsedTokenAccount?.name
    ? parsedTokenAccount.name
    : tokenName
    ? tokenName
    : "";
  const explorerAddress = isNative
    ? null
    : chainId === CHAIN_ID_ETH
    ? `https://${
        CLUSTER === "testnet" ? "goerli." : ""
      }etherscan.io/address/${useableAddress}`
    : chainId === CHAIN_ID_ETHEREUM_ROPSTEN
    ? `https://${
        CLUSTER === "testnet" ? "ropsten." : ""
      }etherscan.io/address/${useableAddress}`
    : chainId === CHAIN_ID_BSC
    ? `https://${
        CLUSTER === "testnet" ? "testnet." : ""
      }bscscan.com/address/${useableAddress}`
    : chainId === CHAIN_ID_POLYGON
    ? `https://${
        CLUSTER === "testnet" ? "mumbai." : ""
      }polygonscan.com/address/${useableAddress}`
    : chainId === CHAIN_ID_AVAX
    ? `https://${
        CLUSTER === "testnet" ? "testnet." : ""
      }snowtrace.io/address/${useableAddress}`
    : chainId === CHAIN_ID_OASIS
    ? `https://${
        CLUSTER === "testnet" ? "testnet." : ""
      }explorer.emerald.oasis.dev/address/${useableAddress}`
    : chainId === CHAIN_ID_AURORA
    ? `https://${
        CLUSTER === "testnet" ? "testnet." : ""
      }aurorascan.dev/address/${useableAddress}`
    : chainId === CHAIN_ID_FANTOM
    ? `https://${
        CLUSTER === "testnet" ? "testnet." : ""
      }ftmscan.com/address/${useableAddress}`
    : chainId === CHAIN_ID_SOLANA
    ? `https://explorer.solana.com/address/${useableAddress}${
        CLUSTER === "testnet"
          ? "?cluster=devnet"
          : CLUSTER === "devnet"
          ? "?cluster=custom&customUrl=http%3A%2F%2Flocalhost%3A8899"
          : ""
      }`
    : chainId === CHAIN_ID_TERRA
    ? `https://finder.terra.money/${
        CLUSTER === "devnet"
          ? "localterra"
          : CLUSTER === "testnet"
          ? "bombay-12"
          : "columbus-5"
      }/address/${useableAddress}`
    : undefined;
  const explorerName = getExplorerName(chainId);

  const copyToClipboard = useCopyToClipboard(useableAddress);

  const explorerButton = !explorerAddress ? null : (
    <Button
      size="small"
      variant="outlined"
      startIcon={<OpenInNew />}
      className={classes.buttons}
      href={explorerAddress}
      target="_blank"
      rel="noopener noreferrer"
    >
      {"View on " + explorerName}
    </Button>
  );
  //TODO add icon here
  const copyButton = isNative ? null : (
    <Button
      size="small"
      variant="outlined"
      startIcon={<FileCopy />}
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
      {extraContent ? extraContent : null}
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
        className={clsx(classes.mainTypog, {
          [classes.noGutter]: noGutter,
          [classes.noUnderline]: noUnderline,
        })}
        component="div"
      >
        {useableSymbol || addressShort}
      </Typography>
    </StyledTooltip>
  );
}
