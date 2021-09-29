import {
  ChainId,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
} from "@certusone/wormhole-sdk";
import { Button, makeStyles, Tooltip, Typography } from "@material-ui/core";
import { FileCopy, OpenInNew } from "@material-ui/icons";
import { withStyles } from "@material-ui/styles";
import clsx from "clsx";
import useCopyToClipboard from "../hooks/useCopyToClipboard";
import { CLUSTER } from "../utils/consts";
import { shortenAddress } from "../utils/solana";

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
  address,
  symbol,
  tokenName,
  variant,
  noGutter,
  noUnderline,
}: {
  chainId: ChainId;
  address?: string;
  logo?: string;
  tokenName?: string;
  symbol?: string;
  variant?: any;
  noGutter?: boolean;
  noUnderline?: boolean;
}) {
  const classes = useStyles();
  const useableAddress = address || "";
  const useableSymbol = symbol || "";
  const isNative = false;
  const addressShort = shortenAddress(useableAddress) || "";

  const useableName = tokenName || "";
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

  const copyToClipboard = useCopyToClipboard(useableAddress);

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
