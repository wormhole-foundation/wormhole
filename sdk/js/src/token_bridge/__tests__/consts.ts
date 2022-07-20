import { describe, expect, it } from "@jest/globals";
import { Connection, PublicKey } from "@solana/web3.js";

const ci = !!process.env.CI;

// see devnet.md
export const ETH_NODE_URL = ci ? "ws://eth-devnet:8545" : "ws://localhost:8545";
export const ETH_PRIVATE_KEY =
  "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d"; // account 0
// account 1 used by NFT tests
export const ETH_PRIVATE_KEY2 =
  "0x6370fd033278c143179d81c5526140625662b8daa446c22ee2d73db3707e620c"; // account 2
export const ETH_PRIVATE_KEY3 =
  "0x646f1ce2fdad0e6deeeb5c7e8e5543bdde65e86029e2fd9fc169899c440a7913"; // account 3
export const ETH_PRIVATE_KEY4 =
  "0xadd53f9a7e588d003326d1cbf9e4a43c061aadd9bc938c843a79e7b4fd2ad743"; // account 4
export const ETH_PRIVATE_KEY5 =
  "0x395df67f0c2d2d9fe1ad08d1bc8b6627011959b79c53d7dd6a3536a33ab8a4fd"; // account 5
export const ETH_PRIVATE_KEY6 =
  "0xe485d098507f54e7733a205420dfddbe58db035fa577fc294ebd14db90767a52"; // account 6
export const ETH_PRIVATE_KEY7 =
  "0xa453611d9419d0e56f499079478fd72c37b251a94bfde4d19872c44cf65386e3"; // account 7
export const ETH_PRIVATE_KEY8 =
  "0x829e924fdf021ba3dbbc4225edfece9aca04b929d6e75613329ca6f1d31c0bb4"; // account 8
export const ETH_PRIVATE_KEY9 =
  "0xb0057716d5917badaf911b193b12b910811c1497b5bada8d7711f758981c3773"; // account 9
export const SOLANA_HOST = ci
  ? "http://solana-devnet:8899"
  : "http://localhost:8899";
export const SOLANA_PRIVATE_KEY = new Uint8Array([
  14, 173, 153, 4, 176, 224, 201, 111, 32, 237, 183, 185, 159, 247, 22, 161, 89,
  84, 215, 209, 212, 137, 10, 92, 157, 49, 29, 192, 101, 164, 152, 70, 87, 65,
  8, 174, 214, 157, 175, 126, 98, 90, 54, 24, 100, 177, 247, 77, 19, 112, 47,
  44, 165, 109, 233, 102, 14, 86, 109, 29, 134, 145, 132, 141,
]);
export const TERRA_NODE_URL = ci
  ? "http://terra-terrad:1317"
  : "http://localhost:1317";
export const TERRA_CHAIN_ID = "localterra";
export const TERRA_GAS_PRICES_URL = ci
  ? "http://terra-fcd:3060/v1/txs/gas_prices"
  : "http://localhost:3060/v1/txs/gas_prices";
export const TERRA2_GAS_PRICES_URL = ci
  ? "http://terra2-fcd:3060/v1/txs/gas_prices"
  : "http://localhost:3061/v1/txs/gas_prices";
export const TERRA_PRIVATE_KEY =
  "notice oak worry limit wrap speak medal online prefer cluster roof addict wrist behave treat actual wasp year salad speed social layer crew genius";
export const TERRA2_PRIVATE_KEY =
  "symbol force gallery make bulk round subway violin worry mixture penalty kingdom boring survey tool fringe patrol sausage hard admit remember broken alien absorb"; // test3
export const TEST_ERC20 = "0x2D8BE6BF0baA74e0A907016679CaE9190e80dD0A";
export const TEST_SOLANA_TOKEN = "2WDq7wSs9zYrpx2kbHDA4RUTRch2CCTP6ZWaH4GNfnQQ";
export const WORMHOLE_RPC_HOSTS = ci
  ? ["http://guardian:7071"]
  : ["http://localhost:7071"];

export type Environment = "devnet" | "testnet" | "mainnet";
export const CLUSTER: Environment = "devnet" as Environment; //This is the currently selected environment.

export const TERRA_HOST =
  CLUSTER === "mainnet"
    ? {
        URL: "https://lcd.terra.dev",
        chainID: "columbus-5",
        name: "mainnet",
        isClassic: true,
      }
    : CLUSTER === "testnet"
    ? {
        URL: "https://bombay-lcd.terra.dev",
        chainID: "bombay-12",
        name: "testnet",
        isClassic: true,
      }
    : {
        URL: TERRA_NODE_URL,
        chainID: "columbus-5",
        name: "localterra",
        isClassic: true,
      };

describe("consts should exist", () => {
  it("has Solana test token", () => {
    expect.assertions(1);
    const connection = new Connection(SOLANA_HOST, "confirmed");
    return expect(
      connection.getAccountInfo(new PublicKey(TEST_SOLANA_TOKEN))
    ).resolves.toBeTruthy();
  });
});
