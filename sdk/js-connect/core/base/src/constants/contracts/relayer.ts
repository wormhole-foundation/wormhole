import { MapLevel } from "../../utils";
import { Network } from "../networks";
import { Chain } from "../chains";

export const relayerContracts = [[
  "Mainnet", [
    ["Ethereum",  "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911"],
    ["Bsc",       "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911"],
    ["Polygon",   "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911"],
    ["Avalanche", "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911"],
    ["Fantom",    "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911"],
    ["Klaytn",    "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911"],
    ["Celo",      "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911"],
    ["Acala",     "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911"],
    ["Karura",    "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911"],
    ["Moonbeam",  "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911"],
    ["Base",      "0x706f82e9bb5b0813501714ab5974216704980e31"],
    ["Arbitrum",  "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911"],
    ["Optimism",  "0x27428DD2d3DD32A4D7f7C497eAaa23130d894911"],
  ]], [
  "Testnet", [
    ["Ethereum",  "0x28D8F1Be96f97C1387e94A53e00eCcFb4E75175a"],
    ["Bsc",       "0x80aC94316391752A193C1c47E27D382b507c93F3"],
    ["Polygon",   "0x0591C25ebd0580E0d4F27A82Fc2e24E7489CB5e0"],
    ["Avalanche", "0xA3cF45939bD6260bcFe3D66bc73d60f19e49a8BB"],
    ["Fantom",    "0x9563a59c15842a6f322b10f69d1dd88b41f2e97b"],
    ["Celo",      "0x306B68267Deb7c5DfCDa3619E22E9Ca39C374f84"],
    ["Moonbeam",  "0x0591C25ebd0580E0d4F27A82Fc2e24E7489CB5e0"],
    ["Arbitrum",  "0xAd753479354283eEE1b86c9470c84D42f229FF43"],
    ["Optimism",  "0x01A957A525a5b7A72808bA9D10c389674E459891"],
    ["Base",      "0xea8029CD7FCAEFFcD1F53686430Db0Fc8ed384E1"],
  ]], [
  "Devnet", [
    ["Ethereum",  "0xE66C1Bc1b369EF4F376b84373E3Aa004E8F4C083"],
    ["Bsc",       "0xE66C1Bc1b369EF4F376b84373E3Aa004E8F4C083"],
  ]],
] as const satisfies MapLevel<Network, MapLevel<Chain, string>>;
