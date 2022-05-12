import { ChainId, CHAIN_ID_ETH } from "@certusone/wormhole-sdk";
import {
  Button,
  makeStyles,
  Typography,
  useMediaQuery,
  useTheme,
} from "@material-ui/core";
import { ArrowForward } from "@material-ui/icons";
import { useCallback, useMemo, useState } from "react";
import { NotionalTVL } from "../../../hooks/useTVL";
import { ChainInfo, getChainShortName } from "../../../utils/consts";
import { createChainTVLChartData, formatTVL } from "./utils";

const useStyles = makeStyles(() => ({
  table: {
    borderSpacing: "16px",
    overflowX: "auto",
    display: "block",
  },
  button: {
    height: "30px",
    textTransform: "none",
    width: "150px",
    fontSize: "12px",
  },
}));

const TVLBarChart = ({
  tvl,
  onChainSelected,
}: {
  tvl: NotionalTVL;
  onChainSelected: (chainInfo: ChainInfo) => void;
}) => {
  const classes = useStyles();

  const [mouseOverChainId, setMouseOverChainId] =
    useState<ChainId>(CHAIN_ID_ETH);

  const chainTVLs = useMemo(() => {
    return createChainTVLChartData(tvl);
  }, [tvl]);

  const handleClick = useCallback(
    (chainInfo: ChainInfo) => {
      onChainSelected(chainInfo);
    },
    [onChainSelected]
  );

  const handleMouseOver = useCallback((chainId: ChainId) => {
    setMouseOverChainId(chainId);
  }, []);

  const theme = useTheme();
  const isSmall = useMediaQuery(theme.breakpoints.down("sm"));

  return (
    <table className={classes.table}>
      <tbody>
        {chainTVLs.map((chainTVL) => (
          <tr
            key={chainTVL.chainInfo.id}
            onMouseOver={() => handleMouseOver(chainTVL.chainInfo.id)}
          >
            <td style={{ textAlign: "right" }}>
              <Typography noWrap display="inline">
                {getChainShortName(chainTVL.chainInfo.id)}
              </Typography>
            </td>
            <td>
              <img
                src={chainTVL.chainInfo.logo}
                alt={""}
                width={24}
                height={24}
              />
            </td>
            <td width="100%">
              <div
                style={{
                  height: 30,
                  width: `${chainTVL.tvlRatio}%`,
                  backgroundImage:
                    "linear-gradient(90deg, #F44B1B 0%, #EEB430 100%)",
                }}
              ></div>
            </td>
            <td>
              <Typography noWrap display="inline">
                {formatTVL(chainTVL.tvl)}
              </Typography>
            </td>
            <td>
              {isSmall || mouseOverChainId === chainTVL.chainInfo.id ? (
                <Button
                  variant="outlined"
                  endIcon={<ArrowForward />}
                  onClick={() => handleClick(chainTVL.chainInfo)}
                  className={classes.button}
                >
                  View assets
                </Button>
              ) : (
                <div style={{ width: 150 }} />
              )}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
};

export default TVLBarChart;
