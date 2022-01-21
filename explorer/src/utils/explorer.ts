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
  }

  if (base) {
    return `${base}${transactionId}`;
  }
  return "";
};

const chainColors: { [chain: string]: string } = {
  "*": "hsl(183, 100%, 61%)",
  "1": "hsl(297, 100%, 61%)",
  "2": "hsl(235, 5%, 43%)",
  "3": "hsl(235, 100%, 61%)",
  "4": "hsl(54, 100%, 61%)",
  "5": "hsl(271, 100%, 61%)",
  "6": "hsl(360, 100%, 61%)",
  "7": "hsl(204, 100%, 48%",
};

export {
  makeDate,
  makeGroupName,
  chainColors,
  truncateAddress,
  contractNameFormatter,
  nativeExplorerContractUri,
  nativeExplorerTxUri,
  getNativeAddress,
};
