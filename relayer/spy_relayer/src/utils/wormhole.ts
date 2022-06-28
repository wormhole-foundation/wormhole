import { ChainId } from "@certusone/wormhole-sdk";
import {BigNumber} from "ethers";

export const chainIDStrings: { [key in ChainId]: string } = {
  0: "Unset",
  1: "Solana",
  2: "Ethereum",
  3: "Terra",
  4: "BSC",
  5: "Polygon",
  6: "Avalanche",
  7: "Oasis",
  8: "Algorand",
  9: "Aurora",
  10: "Fantom",
  11: "Karura",
  12: "Acala",
  13: "Klaytn",
  14: "Celo",
  15: "NEAR",
  16: "Moonbeam",
  17: "Neon",
  10001: "Ropsten",
};

export const parseTransferPayload = function (arr) { return ({
  amount: BigNumber.from(arr.slice(1, 1 + 32)).toBigInt(),
  originAddress: arr.slice(33, 33 + 32).toString("hex"),
  originChain: arr.readUInt16BE(65),
  targetAddress: arr.slice(67, 67 + 32).toString("hex"),
  targetChain: arr.readUInt16BE(99),
  fromAddress: arr.slice(101, 101 + 32).toString("hex").toLowerCase()
});
}





