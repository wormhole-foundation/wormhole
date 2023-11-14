import { MapLevel } from "../../utils";
import { Network } from "../networks";
import { Chain } from "../chains";

// Some chains are required to post proof of their blocks to other chains
// and the transaction containing that proof must be finalized
// before a transaction contained in one of those blocks is considered final
export const rollupContractAddresses = [[
  "Mainnet", [
    ["Polygon",  ["Ethereum", "0x86E4Dc95c7FBdBf52e33D563BbDB00823894C287"]],
    ["Optimism", ["Ethereum", "0xdfe97868233d1aa22e815a266982f2cf17685a27"]],
    ["Arbitrum", ["Ethereum", "0x1c479675ad559dc151f6ec7ed3fbf8cee79582b6"]],
  ]], [
  "Testnet", [
    ["Polygon",  ["Ethereum", "0x2890ba17efe978480615e330ecb65333b880928e"]],
    ["Optimism", ["Ethereum", "0xe6dfba0953616bacab0c9a8ecb3a9bba77fc15c0"]],
    ["Arbitrum", ["Ethereum", "0x45af9ed1d03703e480ce7d328fb684bb67da5049"]], // TODO double check
  ]],
] as const satisfies MapLevel<Network, MapLevel<Chain, readonly [Chain, string]>>;
