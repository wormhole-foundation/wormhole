// TODO: This doesn't belong under wormchain. Move it!
import { ethers } from "ethers";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import axios from "axios";

import {
  CHAIN_ID_ETH,
  CONTRACTS,
  getEmitterAddressEth,
  getSignedVAAWithRetry,
  parseSequenceFromLogEth,
} from "@certusone/wormhole-sdk";

const ci = false; //!!process.env.CI;

const ETH_NODE_URL = ci ? "http://eth-devnet:8545" : "http://localhost:8545";

const ETH_PRIVATE_KEY9 =
  "0xb0057716d5917badaf911b193b12b910811c1497b5bada8d7711f758981c3773";

const GUARDIAN_HOST = ci ? "guardian" : "localhost";
const GUARDIAN_RPCS = [`http://${GUARDIAN_HOST}:7071`];

const CCL_CONTRACT_ADDR = "0x6A4B4A882F5F0a447078b4Fd0b4B571A82371ec2";
const TEST_CONTRACT_LATEST_ADDR = "0xF898012A9c50f4994447d2E8F0844329dD3c9F33";
const TEST_CONTRACT_SAFE_ADDR = "0x7e1E7Ed0F36F62950C8f301C8d653a53aF5FC8c6";
const TEST_CONTRACT_FINALIZED_ADDR =
  "0x10543afEb5c3eb4aE768e03a2D28f7d67B922298";

const TEST_CONTRACT_ABI = [
  {
    type: "function",
    name: "configure",
    inputs: [
      { name: "_consistencyLevel", type: "uint8", internalType: "uint8" },
      { name: "_blocks", type: "uint16", internalType: "uint16" },
    ],
    outputs: [],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "publishMessage",
    inputs: [{ name: "str", type: "string", internalType: "string" }],
    outputs: [{ name: "sequence", type: "uint64", internalType: "uint64" }],
    stateMutability: "payable",
  },
];

const CCL_CONTRACT_ABI = [
  {
    type: "function",
    name: "getConfiguration",
    inputs: [
      { name: "emitterAddress", type: "address", internalType: "address" },
    ],
    outputs: [{ name: "", type: "bytes32", internalType: "bytes32" }],
    stateMutability: "view",
  },
];

// Waiting for finalized can take a while!
jest.setTimeout(180000);

let ethProvider: ethers.providers.JsonRpcProvider;
let ethSigner: ethers.Wallet;

let cclContract: ethers.Contract;

// We use three separate test contracts because the guardian caches the config parameter,
// so even if we update the contract, the guardian probably won't see the change.
let testContractLatest: ethers.Contract;
let testContractSafe: ethers.Contract;
let testContractFinalized: ethers.Contract;

const numBlocks = 5;

beforeAll(async () => {
  // 1. create a signer for Eth
  ethProvider = new ethers.providers.JsonRpcProvider(ETH_NODE_URL);
  ethSigner = new ethers.Wallet(ETH_PRIVATE_KEY9, ethProvider);

  // 1. Create an instance of the custom consistency contract so we can read the config.
  cclContract = new ethers.Contract(
    CCL_CONTRACT_ADDR,
    CCL_CONTRACT_ABI,
    ethProvider
  );

  // 3. Create an instance of the test contract for latest.
  testContractLatest = new ethers.Contract(
    TEST_CONTRACT_LATEST_ADDR,
    TEST_CONTRACT_ABI,
    ethSigner
  );

  // 4. Create an instance of the test contract for safe.
  testContractSafe = new ethers.Contract(
    TEST_CONTRACT_SAFE_ADDR,
    TEST_CONTRACT_ABI,
    ethSigner
  );

  // 5. Create an instance of the test contract for finalized.
  testContractFinalized = new ethers.Contract(
    TEST_CONTRACT_FINALIZED_ADDR,
    TEST_CONTRACT_ABI,
    ethSigner
  );
});

const setCustomConsistencyLevel = async (
  contract: ethers.Contract,
  consistencyLevel: number,
  blocks: number
): Promise<void> => {
  // Call the write function
  const transaction = await contract.configure(consistencyLevel, blocks);

  // Wait for the transaction to be mined
  return transaction.wait();
};

const getCustomConsistencyLevel = async (
  contractAddr: string
): Promise<string> => {
  return cclContract.getConfiguration(contractAddr);
};

const getBlockNumber = async (tag: string): Promise<number> => {
  const str: string = (
    await axios.post(ETH_NODE_URL, {
      jsonrpc: "2.0",
      id: 1,
      method: "eth_getBlockByNumber",
      params: [tag, false],
    })
  ).data?.result?.number;
  return Number(str);
};

describe("Custom Consistency Level Tests", () => {
  test("1. Set and get consistency level", async () => {
    await setCustomConsistencyLevel(testContractLatest, 200, 7);
    expect(await getCustomConsistencyLevel(TEST_CONTRACT_LATEST_ADDR)).toEqual(
      "0x01c8000700000000000000000000000000000000000000000000000000000000"
    );

    // Put the expected values back so we don't break other tests.
    await setCustomConsistencyLevel(testContractLatest, 200, 5);
    expect(await getCustomConsistencyLevel(TEST_CONTRACT_LATEST_ADDR)).toEqual(
      "0x01c8000500000000000000000000000000000000000000000000000000000000"
    );
  });

  test("2. Post a message with latest", async () => {
    // Make sure the config is what we expect.
    expect(await getCustomConsistencyLevel(TEST_CONTRACT_LATEST_ADDR)).toEqual(
      "0x01c8000500000000000000000000000000000000000000000000000000000000"
    );

    // Publish a message.
    const transaction = await testContractLatest.publishMessage(
      "Hello, World!"
    );

    // Wait for the transaction to be mined.
    const receipt = await transaction.wait();

    // Get the block number of the mined transaction.
    const blockNumber: number = Number(receipt.blockNumber as string);

    // Get the sequence from the logs (needed to fetch the vaa).
    const sequence = parseSequenceFromLogEth(
      receipt,
      CONTRACTS.DEVNET.ethereum.core
    );

    // Wait for the VAA to be published.
    await getSignedVAAWithRetry(
      GUARDIAN_RPCS,
      CHAIN_ID_ETH,
      getEmitterAddressEth(TEST_CONTRACT_LATEST_ADDR),
      sequence,
      {
        transport: NodeHttpTransport(),
      }
    );

    // Make sure the VAA wasn't published early. This won't be exact, but it definitely shouldn't be sooner than expected.
    const currentBlockNum = await getBlockNumber("latest");
    console.log(
      "Latest: original block: ",
      blockNumber,
      ", currentBlock: ",
      currentBlockNum
    );
    expect(blockNumber + numBlocks).toBeLessThanOrEqual(currentBlockNum);
  });

  test("3. Post a message with safe", async () => {
    // Make sure the config is what we expect.
    expect(await getCustomConsistencyLevel(TEST_CONTRACT_SAFE_ADDR)).toEqual(
      "0x01c9000500000000000000000000000000000000000000000000000000000000"
    );

    // Publish a message.
    const transaction = await testContractSafe.publishMessage("Hello, World!");

    // Wait for the transaction to be mined.
    const receipt = await transaction.wait();

    // Get the block number of the mined transaction.
    const blockNumber: number = Number(receipt.blockNumber as string);

    // Get the sequence from the logs (needed to fetch the vaa).
    const sequence = parseSequenceFromLogEth(
      receipt,
      CONTRACTS.DEVNET.ethereum.core
    );

    // Wait for the VAA to be published.
    await getSignedVAAWithRetry(
      GUARDIAN_RPCS,
      CHAIN_ID_ETH,
      getEmitterAddressEth(TEST_CONTRACT_SAFE_ADDR),
      sequence,
      {
        transport: NodeHttpTransport(),
      }
    );

    // Make sure the VAA wasn't published early. This won't be exact, but it definitely shouldn't be sooner than expected.
    const currentLatest = await getBlockNumber("latest");
    const currentSafe = await getBlockNumber("safe");
    console.log(
      "Safe: original block: ",
      blockNumber,
      ", currentLatest: ",
      currentLatest,
      ", currentSafe: ",
      currentSafe
    );
    expect(blockNumber).toBeLessThanOrEqual(currentSafe);
    expect(blockNumber + numBlocks).toBeLessThanOrEqual(currentLatest);
  });

  test("4. Post a message with finalized", async () => {
    // Make sure the config is what we expect.
    expect(
      await getCustomConsistencyLevel(TEST_CONTRACT_FINALIZED_ADDR)
    ).toEqual(
      "0x01ca000500000000000000000000000000000000000000000000000000000000"
    );

    // Publish a message.
    const transaction = await testContractFinalized.publishMessage(
      "Hello, World!"
    );

    // Wait for the transaction to be mined.
    const receipt = await transaction.wait();

    // Get the block number of the mined transaction.
    const blockNumber: number = Number(receipt.blockNumber as string);

    // Get the sequence from the logs (needed to fetch the vaa).
    const sequence = parseSequenceFromLogEth(
      receipt,
      CONTRACTS.DEVNET.ethereum.core
    );

    // Wait for the VAA to be published.
    await getSignedVAAWithRetry(
      GUARDIAN_RPCS,
      CHAIN_ID_ETH,
      getEmitterAddressEth(TEST_CONTRACT_FINALIZED_ADDR),
      sequence,
      {
        transport: NodeHttpTransport(),
      }
    );

    // Make sure the VAA wasn't published early. This won't be exact, but it definitely shouldn't be sooner than expected.
    const currentLatest = await getBlockNumber("latest");
    const currentFinalized = await getBlockNumber("finalized");
    console.log(
      "Finalized: original block: ",
      blockNumber,
      ", currentLatest: ",
      currentLatest,
      ", currentFinalized: ",
      currentFinalized
    );
    expect(blockNumber).toBeLessThanOrEqual(currentFinalized);
    expect(blockNumber + numBlocks).toBeLessThanOrEqual(currentLatest);
  });
});
