import { makeStyles, Typography } from "@material-ui/core";
import { CHAINS_BY_ID } from "../../utils/consts";
import SmartAddress from "../SmartAddress";
import { useTargetInfo } from "./Target";

const useStyles = makeStyles((theme) => ({
  description: {
    textAlign: "center",
  },
}));

export default function TargetPreview() {
  const classes = useStyles();
  const {
    targetChain,
    readableTargetAddress,
    targetAsset,
    symbol,
    tokenName,
    logo,
  } = useTargetInfo();

  const explainerContent =
    targetChain && readableTargetAddress ? (
      <>
        {targetAsset ? (
          <>
            <span>and receive</span>
            <SmartAddress
              chainId={targetChain}
              address={targetAsset}
              symbol={symbol}
              tokenName={tokenName}
              logo={logo}
              isAsset
            />
          </>
        ) : null}
        <span>to</span>
        <SmartAddress chainId={targetChain} address={readableTargetAddress} />
        <span>on {CHAINS_BY_ID[targetChain].name}</span>
      </>
    ) : (
      ""
    );

  return (
    <Typography
      component="div"
      variant="subtitle2"
      className={classes.description}
    >
      {explainerContent}
    </Typography>
  );
}
