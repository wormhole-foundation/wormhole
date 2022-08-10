import { describe, expect, it } from "@jest/globals";
import { Connection, PublicKey } from "@solana/web3.js";

// see devnet.md
export const ETH_NODE_URL = "ws://localhost:8545";
export const ETH_PRIVATE_KEY =
  "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d";
export const ETH_PUBLIC_KEY = "0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1";
export const ETH_CORE_BRIDGE_ADDRESS =
  "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550";
export const ETH_TOKEN_BRIDGE_ADDRESS =
  "0x0290FB167208Af455bB137780163b7B7a9a10C16";
export const SOLANA_HOST = "http://localhost:8899";
export const SOLANA_PRIVATE_KEY = new Uint8Array([
  14, 173, 153, 4, 176, 224, 201, 111, 32, 237, 183, 185, 159, 247, 22, 161, 89,
  84, 215, 209, 212, 137, 10, 92, 157, 49, 29, 192, 101, 164, 152, 70, 87, 65,
  8, 174, 214, 157, 175, 126, 98, 90, 54, 24, 100, 177, 247, 77, 19, 112, 47,
  44, 165, 109, 233, 102, 14, 86, 109, 29, 134, 145, 132, 141,
]);
export const SOLANA_CORE_BRIDGE_ADDRESS =
  "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o";
export const SOLANA_TOKEN_BRIDGE_ADDRESS =
  "B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE";
export const TERRA_NODE_URL = "http://localhost:1317";
export const TERRA2_NODE_URL = "http://localhost:1318";
export const TERRA_CHAIN_ID = "localterra";
export const TERRA_GAS_PRICES_URL = "http://localhost:3060/v1/txs/gas_prices";
export const TERRA2_GAS_PRICES_URL = "http://localhost:3061/v1/txs/gas_prices";
export const TERRA_CORE_BRIDGE_ADDRESS =
  "terra18vd8fpwxzck93qlwghaj6arh4p7c5n896xzem5";
export const TERRA_TOKEN_BRIDGE_ADDRESS =
  "terra10pyejy66429refv3g35g2t7am0was7ya7kz2a4";
export const TERRA2_TOKEN_BRIDGE_ADDRESS =
  "terra1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrquka9l6";
export const TERRA_PRIVATE_KEY =
  "notice oak worry limit wrap speak medal online prefer cluster roof addict wrist behave treat actual wasp year salad speed social layer crew genius";
export const TEST_ERC20 = "0x2D8BE6BF0baA74e0A907016679CaE9190e80dD0A";
export const TEST_SOLANA_TOKEN = "2WDq7wSs9zYrpx2kbHDA4RUTRch2CCTP6ZWaH4GNfnQQ";
export const WORMHOLE_RPC_HOSTS = ["http://localhost:7071"];

export const SPY_RELAY_URL = "http://localhost:4201";

describe("consts should exist", () => {
  it("has Solana test token", () => {
    expect.assertions(1);
    const connection = new Connection(SOLANA_HOST, "confirmed");
    return expect(
      connection.getAccountInfo(new PublicKey(TEST_SOLANA_TOKEN))
    ).resolves.toBeTruthy();
  });
});
