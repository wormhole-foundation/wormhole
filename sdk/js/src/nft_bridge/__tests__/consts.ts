import { describe, expect, it } from "@jest/globals";
import { Connection, PublicKey } from "@solana/web3.js";

// see devnet.md
export const ETH_NODE_URL = "ws://localhost:8545";
export const ETH_PRIVATE_KEY =
  "0x6cbed15c793ce57650b9877cf6fa156fbef513c4e6134f022a85b1ffdd59b2a1"; // account 1
export const ETH_CORE_BRIDGE_ADDRESS =
  "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550";
export const ETH_NFT_BRIDGE_ADDRESS =
  "0x26b4afb60d6c903165150c6f0aa14f8016be4aec";
export const SOLANA_HOST = "http://localhost:8899";
export const SOLANA_PRIVATE_KEY = new Uint8Array([
  14, 173, 153, 4, 176, 224, 201, 111, 32, 237, 183, 185, 159, 247, 22, 161, 89,
  84, 215, 209, 212, 137, 10, 92, 157, 49, 29, 192, 101, 164, 152, 70, 87, 65,
  8, 174, 214, 157, 175, 126, 98, 90, 54, 24, 100, 177, 247, 77, 19, 112, 47,
  44, 165, 109, 233, 102, 14, 86, 109, 29, 134, 145, 132, 141,
]);
export const SOLANA_CORE_BRIDGE_ADDRESS =
  "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o";
export const SOLANA_NFT_BRIDGE_ADDRESS =
  "NFTWqJR8YnRVqPDvTJrYuLrQDitTG5AScqbeghi4zSA";
export const TERRA_NODE_URL = "http://localhost:1317";
export const TERRA_CHAIN_ID = "localterra";
export const TERRA_GAS_PRICES_URL = "http://localhost:3060/v1/txs/gas_prices";
export const TERRA_CORE_BRIDGE_ADDRESS =
  "terra18vd8fpwxzck93qlwghaj6arh4p7c5n896xzem5";
export const TERRA_NFT_BRIDGE_ADDRESS =
  "terra19zpyd046u4swqpksr3n44cej4j8pg6ah2y6dcg";
export const TERRA_PRIVATE_KEY =
  "quality vacuum heart guard buzz spike sight swarm shove special gym robust assume sudden deposit grid alcohol choice devote leader tilt noodle tide penalty";
export const TEST_ERC721 = "0x5b9b42d6e4B2e4Bf8d42Eba32D46918e10899B66";
export const TERRA_CW721_CODE_ID = 8;
export const TEST_CW721 = "terra1l425neayde0fzfcv3apkyk4zqagvflm6cmha9v";
export const TEST_SOLANA_TOKEN = "BVxyYhm498L79r4HMQ9sxZ5bi41DmJmeWZ7SCS7Cyvna";
export const WORMHOLE_RPC_HOSTS = ["http://localhost:7071"];

describe("consts should exist", () => {
  it("has Solana test token", () => {
    expect.assertions(1);
    const connection = new Connection(SOLANA_HOST, "confirmed");
    return expect(
      connection.getAccountInfo(new PublicKey(TEST_SOLANA_TOKEN))
    ).resolves.toBeTruthy();
  });
});
