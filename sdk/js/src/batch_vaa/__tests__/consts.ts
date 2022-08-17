import {describe, it} from "@jest/globals";
const batchSenderContract = require("../../../contracts/MockBatchedVAASender.json");

const ci = !!process.env.CI;

// see devnet.md
export const ETH_NODE_URL = ci ? "ws://eth-devnet:8545" : "ws://localhost:8545";
export const ETH_PRIVATE_KEY = "0x6370fd033278c143179d81c5526140625662b8daa446c22ee2d73db3707e620c"; // account 2

// abi and addresses for mock integration contracts
export const MOCK_BATCH_VAA_SENDER_ABI = batchSenderContract.abi;
export const MOCK_BATCH_VAA_SENDER_ADDRESS = "0xf19a2a01b70519f67adb309a994ec8c69a967e8b";

// wormhole event ABIs
export const WORMHOLE_MESSAGE_EVENT_ABI = [
  "event LogMessagePublished(address indexed sender, uint64 sequence, uint32 nonce, bytes payload, uint8 consistencyLevel)",
];

describe("Consts Should Exist", () => {
  it("Dummy Test", () => {
    return;
  });
});
