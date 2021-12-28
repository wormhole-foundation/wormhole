import { Button, makeStyles, Typography } from "@material-ui/core";
import { Launch } from "@material-ui/icons";
import { TokenInfo } from "@solana/spl-token-registry";
import { useSelector } from "react-redux";
import useMarketsMap from "../../hooks/useMarketsMap";
import { DataWrapper } from "../../store/helpers";
import {
  selectSolanaTokenMap,
  selectTransferSourceAsset,
  selectTransferSourceChain,
  selectTransferTargetAsset,
  selectTransferTargetChain,
} from "../../store/selectors";
import { JUPITER_SWAP_BASE_URL } from "../../utils/consts";

const useStyles = makeStyles((theme) => ({
  description: {
    marginTop: theme.spacing(1),
  },
  button: {
    margin: theme.spacing(0.5, 0.5),
  },
}));

function getJupiterSwapUrl(
  link: string,
  targetAsset: string,
  tokenMap: DataWrapper<TokenInfo[]>
) {
  if (!tokenMap.error && !tokenMap.isFetching && tokenMap.data) {
    const tokenInfo = tokenMap.data.find((value) => {
      return value.address === targetAsset;
    });
    if (tokenInfo) {
      const sourceSymbol = tokenInfo.symbol;
      if (sourceSymbol) {
        const targetSymbol = sourceSymbol === "UST" ? "SOL" : "UST";
        return `${JUPITER_SWAP_BASE_URL}/${sourceSymbol}-${targetSymbol}`;
      }
    }
  }
  return link;
}

export default function FeaturedMarkets() {
  const sourceChain = useSelector(selectTransferSourceChain);
  const sourceAsset = useSelector(selectTransferSourceAsset);
  const targetChain = useSelector(selectTransferTargetChain);
  const targetAsset = useSelector(selectTransferTargetAsset);
  const solanaTokenMap = useSelector(selectSolanaTokenMap);
  const { data: marketsData } = useMarketsMap(true);
  const classes = useStyles();

  if (
    !sourceAsset ||
    !targetAsset ||
    !marketsData ||
    !marketsData.markets ||
    !marketsData.tokenMarkets
  ) {
    return null;
  }

  const tokenMarkets =
    marketsData.tokenMarkets[sourceChain]?.[targetChain]?.[sourceAsset];
  if (!tokenMarkets) {
    return null;
  }

  const tokenMarketButtons = [];
  for (const market of tokenMarkets.markets) {
    const marketInfo = marketsData.markets[market];
    if (marketInfo) {
      const url =
        market === "jupiter"
          ? getJupiterSwapUrl(marketInfo.link, sourceAsset, solanaTokenMap)
          : marketInfo.link;
      tokenMarketButtons.push(
        <Button
          key={market}
          size="small"
          variant="outlined"
          color="secondary"
          endIcon={<Launch />}
          href={url}
          target="_blank"
          rel="noopener noreferrer"
          className={classes.button}
        >
          {marketInfo.name}
        </Button>
      );
    }
  }

  return tokenMarketButtons.length ? (
    <div style={{ textAlign: "center" }}>
      <Typography
        variant="subtitle2"
        gutterBottom
        className={classes.description}
      >
        Featured markets
      </Typography>
      {tokenMarketButtons}
    </div>
  ) : null;
}
