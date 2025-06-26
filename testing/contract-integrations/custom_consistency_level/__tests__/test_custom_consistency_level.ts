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

import * as CustomConsistencyLevel from "../../../../ethereum/build-forge/CustomConsistencyLevel.sol/CustomConsistencyLevel.json";
import * as TestCustomConsistencyLevel from "../../../../ethereum/build-forge/TestCustomConsistencyLevel.sol/TestCustomConsistencyLevel.json";

const ci = process.env.CI == "true";

const ETH_NODE_URL = ci ? "http://eth-devnet:8545" : "http://localhost:8545";

const ETH_PRIVATE_KEY14 =
  "0x21d7212f3b4e5332fd465877b64926e3532653e2798a11255a46f533852dfe46";

const GUARDIAN_HOST = ci ? "guardian" : "localhost";
const GUARDIAN_RPCS = [`http://${GUARDIAN_HOST}:7071`];

const CORE_CONTRACT_ADDR = "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550";
const CCL_CONTRACT_ADDR = "0x6A4B4A882F5F0a447078b4Fd0b4B571A82371ec2";

// Waiting for safe and finalized can take a while!
jest.setTimeout(300000);

let ethProvider: ethers.providers.JsonRpcProvider;
let ethSigner: ethers.Wallet;

let cclContract: ethers.Contract;
let testContractFactory: ethers.ContractFactory;

const numBlocks = 5;

beforeAll(async () => {
  // 1. create a signer for Eth
  ethProvider = new ethers.providers.JsonRpcProvider(ETH_NODE_URL);
  ethSigner = new ethers.Wallet(ETH_PRIVATE_KEY14, ethProvider);

  // 1. Connect to the custom consistency contract so we can read the config.
  cclContract = new ethers.Contract(
    CCL_CONTRACT_ADDR,
    CustomConsistencyLevel.abi,
    ethProvider
  );

  // Get the contract factory so we can deploy instances of the test contract.
  testContractFactory = new ethers.ContractFactory(
    TestCustomConsistencyLevel.abi,
    TestCustomConsistencyLevel.bytecode,
    ethSigner
  );
});

const deployTestContract = async (
  consistencyLevel: number,
  blocks: number
): Promise<ethers.Contract> => {
  // Deploy the contract with the specified parameters.
  const contract = await testContractFactory.deploy(
    CORE_CONTRACT_ADDR,
    CCL_CONTRACT_ADDR,
    consistencyLevel,
    blocks
  );

  // Wait for the contract to be deployed and return it.
  await contract.deployTransaction.wait();
  return contract;
};

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
    const testContract = await deployTestContract(200, 42);
    expect(await getCustomConsistencyLevel(testContract.address)).toEqual(
      "0x01c8002a00000000000000000000000000000000000000000000000000000000"
    );

    await setCustomConsistencyLevel(testContract, 200, 7);
    expect(await getCustomConsistencyLevel(testContract.address)).toEqual(
      "0x01c8000700000000000000000000000000000000000000000000000000000000"
    );
  });

  test("2. Post a message with latest", async () => {
    // Create an instance of the test contract for latest.
    const contract = await deployTestContract(200, numBlocks);
    console.log("Latest: deployed to address ", contract.address);

    // Make sure the config is what we expect.
    expect(await getCustomConsistencyLevel(contract.address)).toEqual(
      "0x01c8000500000000000000000000000000000000000000000000000000000000"
    );

    // Publish a message.
    const transaction = await contract.publishMessage("Hello, World!");

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
      getEmitterAddressEth(contract.address),
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
    // Create an instance of the test contract for safe.
    const contract = await deployTestContract(201, numBlocks);
    console.log("Safe: deployed to address ", contract.address);

    // Make sure the config is what we expect.
    expect(await getCustomConsistencyLevel(contract.address)).toEqual(
      "0x01c9000500000000000000000000000000000000000000000000000000000000"
    );

    // Publish a message.
    const transaction = await contract.publishMessage("Hello, World!");

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
      getEmitterAddressEth(contract.address),
      sequence,
      {
        transport: NodeHttpTransport(),
      }
    );

    // Make sure the VAA wasn't published early. This won't be exact, but it definitely shouldn't be sooner than expected.
    const currentSafe = await getBlockNumber("safe");
    console.log(
      "Safe: original block: ",
      blockNumber,
      ", currentSafe: ",
      currentSafe
    );
    expect(blockNumber + numBlocks).toBeLessThanOrEqual(currentSafe);
  });

  test("4. Post a message with finalized", async () => {
    // Create an instance of the test contract for finalized.
    const contract = await deployTestContract(202, numBlocks);
    console.log("Finalized: deployed to address ", contract.address);

    // Make sure the config is what we expect.
    expect(await getCustomConsistencyLevel(contract.address)).toEqual(
      "0x01ca000500000000000000000000000000000000000000000000000000000000"
    );

    // Publish a message.
    const transaction = await contract.publishMessage("Hello, World!");

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
      getEmitterAddressEth(contract.address),
      sequence,
      {
        transport: NodeHttpTransport(),
      }
    );

    // Make sure the VAA wasn't published early. This won't be exact, but it definitely shouldn't be sooner than expected.
    const currentFinalized = await getBlockNumber("finalized");
    console.log(
      "Finalized: original block: ",
      blockNumber,
      ", currentFinalized: ",
      currentFinalized
    );
    expect(blockNumber + numBlocks).toBeLessThanOrEqual(currentFinalized);
  });

  test("5. Post a message with latest, no additional blocks", async () => {
    // Create an instance of the test contract for latest.
    const contract = await deployTestContract(200, 0);
    console.log("Latest0: deployed to address ", contract.address);

    // Make sure the config is what we expect.
    expect(await getCustomConsistencyLevel(contract.address)).toEqual(
      "0x01c8000000000000000000000000000000000000000000000000000000000000"
    );

    // Publish a message.
    const transaction = await contract.publishMessage("Hello, World!");

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
      getEmitterAddressEth(contract.address),
      sequence,
      {
        transport: NodeHttpTransport(),
      }
    );

    // Make sure the VAA wasn't published early. This won't be exact, but it definitely shouldn't be sooner than expected.
    const currentBlockNum = await getBlockNumber("latest");
    console.log(
      "Latest0: original block: ",
      blockNumber,
      ", currentBlock: ",
      currentBlockNum
    );
    expect(blockNumber).toBeLessThanOrEqual(currentBlockNum);
  });
});
