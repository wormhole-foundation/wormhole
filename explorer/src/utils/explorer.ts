import {
  ChainId,
  CHAIN_ID_TERRA,
  hexToNativeString,
  isEVMChain,
} from "@certusone/wormhole-sdk";
import { fromHex } from "@cosmjs/encoding";
import { PublicKey } from "@solana/web3.js";
import { ActiveNetwork, useNetworkContext } from "../contexts/NetworkContext";
import { chainEnums, ChainID, chainIDs } from "./consts";

const makeDate = (date: string): string => {
  const [_, month, day] = date.split("-");
  if (!month || !day) {
    throw Error("Invalid date supplied to makeDate. Expects YYYY-MM-DD.");
  }
  return `${month}/${day}`;
};
const makeGroupName = (
  groupKey: string,
  activeNetwork: ActiveNetwork,
  emitterChain?: number
): string => {
  let ALL = "All Wormhole messages";
  if (emitterChain) {
    ALL = `All ${chainEnums[emitterChain]} messages`;
  }
  let group = groupKey === "*" ? ALL : groupKey;
  if (group.includes(":")) {
    // subKey is chainID:addresss
    let parts = groupKey.split(":");
    group = `${ChainID[Number(parts[0])]} ${contractNameFormatter(
      parts[1],
      Number(parts[0]),
      activeNetwork
    )}`;
  } else if (group != ALL) {
    // subKey is a chainID
    group = ChainID[Number(groupKey)];
  }
  return group;
};

const getNativeAddress = (
  chainId: number,
  emitterAddress: string,
  activeNetwork?: ActiveNetwork
): string => {
  let nativeAddress = "";

  if (isEVMChain(chainId as ChainId)) {
    // remove zero-padding
    let unpadded = emitterAddress.slice(-40);
    nativeAddress = `0x${unpadded}`.toLowerCase();
  } else if (chainId === chainIDs["terra"]) {
    nativeAddress = (
      hexToNativeString(emitterAddress, CHAIN_ID_TERRA) || ""
    ).toLowerCase();
  } else if (chainId === chainIDs["solana"]) {
    if (!activeNetwork) {
      activeNetwork = useNetworkContext().activeNetwork;
    }
    const chainName = chainEnums[chainId].toLowerCase();

    // use the "chains" map of hex: nativeAdress first
    if (emitterAddress in activeNetwork.chains[chainName]) {
      let desc = activeNetwork.chains[chainName][emitterAddress];
      if (desc in activeNetwork.chains[chainName]) {
        // lookup the contract address
        nativeAddress = activeNetwork.chains[chainName][desc];
      }
    } else {
      let hex = fromHex(emitterAddress);
      let pubKey = new PublicKey(hex);
      nativeAddress = pubKey.toString();
    }
  }
  return nativeAddress;
};

const truncateAddress = (address: string): string => {
  return `${address.slice(0, 4)}...${address.slice(-4)}`;
};

const contractNameFormatter = (
  address: string,
  chainId: number,
  activeNetwork?: ActiveNetwork
): string => {
  if (!activeNetwork) {
    activeNetwork = useNetworkContext().activeNetwork;
  }

  const chainName = chainEnums[chainId].toLowerCase();
  let nativeAddress = getNativeAddress(chainId, address, activeNetwork);

  let truncated = truncateAddress(nativeAddress || address);
  let formatted = truncated;

  if (nativeAddress in activeNetwork.chains[chainName]) {
    // add the description of the contract, if we know it
    let desc = activeNetwork.chains[chainName][nativeAddress];
    formatted = `${desc} (${truncated})`;
  }
  return formatted;
};

const nativeExplorerContractUri = (
  chainId: number,
  address: string,
  activeNetwork?: ActiveNetwork
): string => {
  if (!activeNetwork) {
    activeNetwork = useNetworkContext().activeNetwork;
  }

  const nativeAddress = getNativeAddress(chainId, address, activeNetwork);
  if (nativeAddress) {
    let base = "";
    if (chainId === chainIDs["solana"]) {
      base = "https://explorer.solana.com/address/";
    } else if (chainId === chainIDs["ethereum"]) {
      base = "https://etherscan.io/address/";
    } else if (chainId === chainIDs["terra"]) {
      base = "https://finder.terra.money/columbus-5/address/";
    } else if (chainId === chainIDs["bsc"]) {
      base = "https://bscscan.com/address/";
    } else if (chainId === chainIDs["polygon"]) {
      base = "https://polygonscan.com/address/";
    } else if (chainId === chainIDs["avalanche"]) {
      base = "https://snowtrace.io/address/";
    } else if (chainId === chainIDs["oasis"]) {
      base = "https://explorer.oasis.updev.si/address/";
    } else if (chainId === chainIDs["fantom"]) {
      base = "https://ftmscan.com/address/";
    } else if (chainId === chainIDs["aurora"]) {
      base = "https://aurorascan.dev/address/";
    }
    return `${base}${nativeAddress}`;
  }
  return "";
};
const nativeExplorerTxUri = (
  chainId: number,
  transactionId: string
): string => {
  let base = "";
  if (chainId === chainIDs["solana"]) {
    base = "https://explorer.solana.com/address/";
  } else if (chainId === chainIDs["ethereum"]) {
    base = "https://etherscan.io/tx/";
  } else if (chainId === chainIDs["terra"]) {
    base = "https://finder.terra.money/columbus-5/tx/";
  } else if (chainId === chainIDs["bsc"]) {
    base = "https://bscscan.com/tx/";
  } else if (chainId === chainIDs["polygon"]) {
    base = "https://polygonscan.com/tx/";
  } else if (chainId === chainIDs["avalanche"]) {
    base = "https://snowtrace.io/tx/";
  } else if (chainId === chainIDs["oasis"]) {
    base = "https://explorer.emerald.oasis.dev/tx/";
  } else if (chainId === chainIDs["fantom"]) {
    base = "https://ftmscan.com/tx/";
  } else if (chainId === chainIDs["aurora"]) {
    base = "https://aurorascan.dev/tx/";
  }

  if (base) {
    return `${base}${transactionId}`;
  }
  return "";
};

// define colors to represent chains in charts/graphs
const chainColors: { [chain: string]: string } = {
  "*": "hsl(183, 100%, 61%)",
  "1": "hsl(297, 100%, 61%)",
  "2": "hsl(235, 5%, 43%)",
  "3": "hsl(235, 100%, 61%)",
  "4": "hsl(54, 100%, 61%)",
  "5": "hsl(271, 100%, 61%)",
  "6": "hsl(360, 100%, 61%)",
  "7": "hsl(204, 100%, 48%)",
  "10": "hsl(220, 78%, 92%)",
};
const chainIdColors = Object.entries(chainColors).reduce<Array<string>>(
  // returns an array of hsl colors, indexed by chainId
  (accum, [chain, color]) => {
    accum[Number(chain) || 0] = color;
    return accum;
  },
  []
);

const amountFormatter = (num: number, decimals?: number): string => {
  let absNum = Math.abs(num);
  if (absNum > 999 && absNum < 1000000) {
    return (num / 1000).toFixed(decimals !== undefined ? decimals : 1) + "K"; // convert to K with 1 decimal for 1000 < 1 million
  } else if (absNum >= 1000000 && absNum < 1000000000) {
    return (num / 1000000).toFixed(decimals !== undefined ? decimals : 0) + "M"; // convert to M for number from > 1 million
  } else if (absNum >= 1000000000) {
    return (
      (num / 1000000000).toFixed(decimals !== undefined ? decimals : 1) + "B"
    ); // convert to B for number from > 1 billion
  }
  return num.toFixed(decimals !== undefined ? decimals : 0); // if value < 1000, nothing to do
};
const usdFormatter = new Intl.NumberFormat("en-US", {
  style: "currency",
  currency: "USD",
});

export {
  amountFormatter,
  chainColors,
  chainIdColors,
  contractNameFormatter,
  getNativeAddress,
  makeDate,
  makeGroupName,
  nativeExplorerContractUri,
  nativeExplorerTxUri,
  truncateAddress,
  usdFormatter,
};
