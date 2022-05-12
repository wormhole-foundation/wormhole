import * as React from "react";
import binanceChainIcon from "../images/bsc.svg";
import ethereumIcon from "../images/eth.svg";
import solanaIcon from "../images//solana.svg";
import terraIcon from "../images/terra.svg";
import polygonIcon from "../images/polygon.svg";
import avalancheIcon from "../images/avalanche.svg";
import oasisIcon from "../images/oasis.svg";
import fantomIcon from "../images/fantom.svg";
import auroraIcon from "../images/aurora.svg";
import karuraIcon from "../images/karura.svg"
import {
  ChainId,
  CHAIN_ID_AVAX,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_OASIS,
  CHAIN_ID_POLYGON,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  CHAIN_ID_FANTOM,
  CHAIN_ID_AURORA,
  CHAIN_ID_KARURA,
} from "@certusone/wormhole-sdk";
import { chainEnums } from "../utils/consts";
import { Box } from "@mui/material";

const chainIdToSrc = {
  [CHAIN_ID_SOLANA]: solanaIcon,
  [CHAIN_ID_ETH]: ethereumIcon,
  [CHAIN_ID_TERRA]: terraIcon,
  [CHAIN_ID_BSC]: binanceChainIcon,
  [CHAIN_ID_POLYGON]: polygonIcon,
  [CHAIN_ID_AVAX]: avalancheIcon,
  [CHAIN_ID_OASIS]: oasisIcon,
  [CHAIN_ID_FANTOM]: fantomIcon,
  [CHAIN_ID_AURORA]: auroraIcon,
  [CHAIN_ID_KARURA]: karuraIcon,
};

const ChainIcon = ({ chainId }: { chainId: ChainId }) =>
  chainIdToSrc[chainId] ? (
    <Box
      sx={{
        display: "flex",
        alignItems: "center",
        px: chainId === CHAIN_ID_ETH ? 0 : 0.25,
        "&:first-of-type": { pl: 0 },
        "&:last-of-type": { pr: 0 },
      }}
    >
      <img
        src={chainIdToSrc[chainId]}
        alt={chainEnums[chainId] || ""}
        style={{ width: 16 }}
      />
    </Box>
  ) : null;

export default ChainIcon;
