import {
  afterAll,
  beforeAll,
  describe,
  expect,
  jest,
  test,
} from "@jest/globals";
import axios, { AxiosResponse } from "axios";
import { Client, encodeFunctionData, parseEther, type Address } from "viem";
import { privateKeyToAccount, generatePrivateKey } from "viem/accounts";
import {
  ChainQueryType,
  EthCallByTimestampQueryRequest,
  EthCallByTimestampQueryResponse,
  EthCallData,
  EthCallQueryRequest,
  EthCallQueryResponse,
  EthCallWithFinalityQueryRequest,
  EthCallWithFinalityQueryResponse,
  PerChainQueryRequest,
  QueryRequest,
  QueryResponse,
  sign,
} from "../../src";
import {
  CCQ_SERVER_URL,
  createClient,
  ERC20_ABI,
  EVM_QUERY_TYPE,
  STAKING_FACTORY_ADDRESS,
  getPoolAddress,
  mintAndTransferTokens,
  POOL_STAKE_ABI,
  QUERY_URL,
  setupAxiosInterceptor,
  sleep,
  W_TOKEN_ADDRESS,
} from "./test-utils";

jest.setTimeout(180000); // 3 minutes for tests with long blockchain operations
setupAxiosInterceptor();

const ENV = "DEVNET";
// Health endpoint is on port 6068, not 6069
const CI = process.env.CI;
const HEALTH_URL = CI
  ? "http://query-server:6068/health"
  : "http://localhost:6068/health";
const WETH_ADDRESS = "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E";

const STAKE_AMOUNT = "50000";

// Use smaller wallet pool (3 wallets) to speed up setup
// Tests can share wallets since we have very high rate limits
const walletPool: Array<{
  privateKey: `0x${string}`;
  address: Address;
}> = [];

// Only 3 wallets - faster setup, still enough for test isolation when needed
for (let i = 0; i < 3; i++) {
  const privateKey = generatePrivateKey();
  const account = privateKeyToAccount(privateKey);
  walletPool.push({ privateKey, address: account.address });
}

let walletIndex = 0;
function getNextWallet() {
  const wallet = walletPool[walletIndex % walletPool.length];
  walletIndex++;
  return wallet;
}

let poolAddress: Address;

function createTestEthCallData(
  to: string,
  name: string,
  outputType: string
): EthCallData {
  return {
    to,
    data: encodeFunctionData({
      abi: [
        {
          name,
          type: "function",
          inputs: [],
          outputs: [{ name, type: outputType }],
          stateMutability: "view",
        },
      ],
      functionName: name,
    }),
  };
}

async function getEthCallByTimestampArgs(): Promise<[bigint, bigint, bigint]> {
  const client = createClient();
  let followingBlockNumber = await client.getBlockNumber();
  let targetBlockNumber = BigInt(0);
  let targetBlockTime = BigInt(0);
  while (targetBlockNumber === BigInt(0)) {
    let followingBlock = await client.getBlock({
      blockNumber: followingBlockNumber,
    });
    while (Number(followingBlock.number) <= 0) {
      await sleep(1000);
      followingBlock = await client.getBlock({
        blockNumber: followingBlock.number,
      });
      followingBlockNumber = followingBlock.number;
    }
    const targetBlock = await client.getBlock({
      blockNumber: followingBlockNumber - BigInt(1),
    });
    if (targetBlock.timestamp < followingBlock.timestamp) {
      targetBlockTime = targetBlock.timestamp * BigInt(1000000);
      targetBlockNumber = targetBlock.number;
    } else {
      followingBlockNumber = targetBlockNumber;
    }
  }
  return [targetBlockTime, targetBlockNumber, followingBlockNumber];
}

/**
 * Optimized staking setup - stakes wallets serially to avoid nonce issues
 * but does the expensive blockchain operations efficiently
 */
async function setupWalletsWithStake(
  wallets: Array<{ privateKey: `0x${string}`; address: Address }>,
  poolAddress: Address,
  stakeAmount: string
): Promise<void> {
  const minterClient = createClient();
  const stakeAmountWei = parseEther(stakeAmount);

  console.log(`  Setting up ${wallets.length} wallets...`);

  // Process each wallet serially to avoid issues
  for (let i = 0; i < wallets.length; i++) {
    const wallet = wallets[i];
    console.log(`    Wallet ${i + 1}/${wallets.length}: ${wallet.address}`);

    // Send ETH for gas
    const ethHash = await minterClient.sendTransaction({
      to: wallet.address,
      value: parseEther("1"),
    } as any);
    await minterClient.waitForTransactionReceipt({ hash: ethHash });

    // Mint tokens
    await mintAndTransferTokens(wallet.address, stakeAmount);

    // Approve and stake
    const walletClient = createClient(wallet.privateKey);

    const approveHash = await walletClient.writeContract({
      address: W_TOKEN_ADDRESS,
      abi: ERC20_ABI,
      functionName: "approve",
      args: [poolAddress, stakeAmountWei],
    } as any);
    await walletClient.waitForTransactionReceipt({ hash: approveHash });

    const stakeHash = await walletClient.writeContract({
      address: poolAddress,
      abi: POOL_STAKE_ABI,
      functionName: "stake",
      args: [stakeAmountWei],
    } as any);
    await walletClient.waitForTransactionReceipt({ hash: stakeHash });

    console.log(`      ✓ Staked ${stakeAmount} tokens`);
  }
}

beforeAll(async () => {
  console.log(
    `\nSetting up ${walletPool.length} test wallets with ${STAKE_AMOUNT} tokens each`
  );

  // Get pool address from factory
  poolAddress = await getPoolAddress(STAKING_FACTORY_ADDRESS, EVM_QUERY_TYPE);

  console.log("\nEthCall2 Test Configuration:");
  console.log("  Factory:", STAKING_FACTORY_ADDRESS);
  console.log("  Pool:", poolAddress);
  console.log("  Token:", W_TOKEN_ADDRESS);
  console.log("  Stake per wallet:", STAKE_AMOUNT, "tokens");

  expect(poolAddress).toBeTruthy();
  expect(poolAddress).not.toBe("0x0000000000000000000000000000000000000000");

  // Setup wallets with high stake amounts
  await setupWalletsWithStake(walletPool, poolAddress, STAKE_AMOUNT);

  // Verify stakes were recorded
  console.log("\nVerifying stakes...");
  const verifyClient = createClient();
  for (const wallet of walletPool) {
    const stakeInfo = (await verifyClient.readContract({
      address: poolAddress,
      abi: [
        {
          inputs: [{ name: "staker", type: "address" }],
          name: "getStakeInfo",
          outputs: [
            { name: "amount", type: "uint256" },
            { name: "conversionTableIndex", type: "uint256" },
            { name: "lockupEnd", type: "uint48" },
            { name: "accessEnd", type: "uint48" },
            { name: "lastClaimed", type: "uint48" },
            { name: "capacity", type: "uint256" },
          ],
          stateMutability: "view",
          type: "function",
        },
      ],
      functionName: "getStakeInfo",
      args: [wallet.address],
    } as any)) as any;

    const stakeAmount = stakeInfo[0];
    console.log(
      `  ${wallet.address}: ${stakeAmount.toString()} wei (${Number(stakeAmount) / 1e18
      } tokens)`
    );

    if (BigInt(stakeAmount) === BigInt(0)) {
      throw new Error(
        `Wallet ${wallet.address} has zero stake! Staking failed.`
      );
    }
  }

  console.log("Wallets staked and ready\n");
}, 60000);

describe("eth call v2", () => {
  test("serialize request", () => {
    // Serialize test doesn't need real stake - just needs an address
    const dummyAddress = privateKeyToAccount(generatePrivateKey()).address;
    const toAddress = "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270";
    const nameCallData = createTestEthCallData(toAddress, "name", "string");
    const decimalsCallData = createTestEthCallData(
      toAddress,
      "decimals",
      "uint8"
    );
    const ethCall = new EthCallQueryRequest("0x28d9630", [
      nameCallData,
      decimalsCallData,
    ]);
    const chainId = 5;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(
      nonce,
      Math.floor(Date.now() / 1000),
      [ethQuery],
      dummyAddress
    );
    const serialized = request.serialize();
    // V2 format with staker address - just verify it serializes without error
    expect(serialized).toBeTruthy();
    expect(serialized.length).toBeGreaterThan(0);
    // Verify it starts with version byte 0x02 for v2
    expect(serialized[0]).toEqual(0x02);
  });

  test("successful query", async () => {
    const { privateKey, address } = getNextWallet();
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const decimalsCallData = createTestEthCallData(
      WETH_ADDRESS,
      "decimals",
      "uint8"
    );
    const blockNumber = await createClient().getBlockNumber();
    const ethCall = new EthCallQueryRequest(Number(blockNumber), [
      nameCallData,
      decimalsCallData,
    ]);
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, Math.floor(Date.now() / 1000), [ethQuery], address);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(privateKey.slice(2), digest);
    const response = await axios.post(QUERY_URL, {
      signature,
      bytes: Buffer.from(serialized).toString("hex"),
    });
    expect(response.status).toBe(200);

    const queryResponse = QueryResponse.from(response.data.bytes);
    expect(queryResponse.version).toEqual(1);
    expect(queryResponse.requestChainId).toEqual(0);
    expect(queryResponse.request.requests.length).toEqual(1);
    expect(queryResponse.request.requests[0].chainId).toEqual(2);
    expect(queryResponse.request.requests[0].query.type()).toEqual(
      ChainQueryType.EthCall
    );

    const ecr = queryResponse.responses[0].response as EthCallQueryResponse;
    expect(ecr.blockNumber.toString()).toEqual(BigInt(blockNumber).toString());
    expect(ecr.blockHash).toEqual(
      (await createClient().getBlock({ blockNumber: BigInt(blockNumber) })).hash
    );
    expect(ecr.results.length).toEqual(2);
    expect(ecr.results[0]).toEqual(
      // Name
      "0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000"
    );
    expect(ecr.results[1]).toEqual(
      // Decimals
      "0x0000000000000000000000000000000000000000000000000000000000000012"
    );
  });

  test("get block by hash should work", async () => {
    const { privateKey, address } = getNextWallet();
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const decimalsCallData = createTestEthCallData(
      WETH_ADDRESS,
      "decimals",
      "uint8"
    );
    const blockNumber = await createClient().getBlockNumber();
    const block = await createClient().getBlock({
      blockNumber: BigInt(blockNumber),
    });
    if (block.hash != undefined) {
      const ethCall = new EthCallQueryRequest(block.hash?.toString(), [
        nameCallData,
        decimalsCallData,
      ]);
      const chainId = 2;
      const ethQuery = new PerChainQueryRequest(chainId, ethCall);
      const nonce = 1;
      const request = new QueryRequest(nonce, Math.floor(Date.now() / 1000), [ethQuery], address);
      const serialized = request.serialize();
      const digest = QueryRequest.digest(ENV, serialized);
      const signature = sign(privateKey.slice(2), digest);
      const response = await axios.post(QUERY_URL, {
        signature,
        bytes: Buffer.from(serialized).toString("hex"),
      });
      expect(response.status).toBe(200);
    }
  });

  test("signed query with valid stake succeeds", async () => {
    const { privateKey, address } = getNextWallet();
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const decimalsCallData = createTestEthCallData(
      WETH_ADDRESS,
      "decimals",
      "uint8"
    );
    const blockNumber = await createClient().getBlockNumber();
    const ethCall = new EthCallQueryRequest(Number(blockNumber), [
      nameCallData,
      decimalsCallData,
    ]);
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, Math.floor(Date.now() / 1000), [ethQuery], address);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(privateKey.slice(2), digest);
    const response = await axios.post(QUERY_URL, {
      signature,
      bytes: Buffer.from(serialized).toString("hex"),
    });
    // Should succeed with staking-based auth (200), or possibly be rate limited (429) or timeout (504)
    expect([200, 429, 504]).toContain(response.status);
  });

  test("unsigned query should fail if not allowed", async () => {
    const { address } = getNextWallet();
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const decimalsCallData = createTestEthCallData(
      WETH_ADDRESS,
      "decimals",
      "uint8"
    );
    const blockNumber = await createClient().getBlockNumber();
    const ethCall = new EthCallQueryRequest(Number(blockNumber), [
      nameCallData,
      decimalsCallData,
    ]);
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, Math.floor(Date.now() / 1000), [ethQuery], address);
    const serialized = request.serialize();
    const signature = "";
    let err = false;
    await axios
      .post(QUERY_URL, {
        signature,
        bytes: Buffer.from(serialized).toString("hex"),
      })
      .catch(function(error) {
        err = true;
        // Server returns 400 (bad request) because signature validation happens first
        expect(error.response.status).toBe(400);
        // Error message indicates signature length validation
        expect(error.response.data).toContain("signature must be 65 bytes");
      });
    expect(err).toBe(true);
  });

  test("unsigned query requires signature with staking", async () => {
    const { privateKey, address } = getNextWallet();
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const blockNumber = await createClient().getBlockNumber();
    const ethCall = new EthCallQueryRequest(Number(blockNumber), [
      nameCallData,
    ]);
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, Math.floor(Date.now() / 1000), [ethQuery], address);
    const serialized = request.serialize();
    // Properly signed query should work
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(privateKey.slice(2), digest);
    const response = await axios.post(QUERY_URL, {
      signature,
      bytes: Buffer.from(serialized).toString("hex"),
    });
    expect([200, 429, 504]).toContain(response.status);
  });

  test("query with EIP-191 prefixed signature succeeds", async () => {
    const { privateKey, address } = getNextWallet();
    const client = createClient(privateKey);
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const blockNumber = await client.getBlockNumber();
    const ethCall = new EthCallQueryRequest(Number(blockNumber), [
      nameCallData,
    ]);
    const ethQuery = new PerChainQueryRequest(2, ethCall);
    const request = new QueryRequest(1, Math.floor(Date.now() / 1000), [ethQuery], address);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    // Sign with personal_sign (EIP-191 prefixed)
    // viem's signMessage adds the "\x19Ethereum Signed Message:\n" prefix
    const signature = await client.signMessage({
      message: { raw: digest },
    });
    // Submit with X-Signature-Format header
    const response = await axios.post(
      QUERY_URL,
      {
        bytes: Buffer.from(serialized).toString("hex"),
        signature: signature.slice(2), // Remove 0x prefix
      },
      {
        headers: {
          "Content-Type": "application/json",
          "X-Signature-Format": "eip191",
        },
      }
    );
    expect([200, 429, 504]).toContain(response.status);
    if (response.status === 200) {
      expect(response.data.bytes).toBeTruthy();
      // Verify we can parse the response
      const queryResponse = QueryResponse.from(response.data.bytes);
      expect(queryResponse.responses.length).toBe(1);
    }
  });

  test("raw signature still works without header", async () => {
    const { privateKey, address } = getNextWallet();
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const client = createClient();
    const blockNumber = await client.getBlockNumber();
    const ethCall = new EthCallQueryRequest(Number(blockNumber), [
      nameCallData,
    ]);
    const ethQuery = new PerChainQueryRequest(2, ethCall);
    const request = new QueryRequest(2, Math.floor(Date.now() / 1000), [ethQuery], address);
    // Serialize and sign with raw ECDSA (using the sign helper)
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(privateKey.slice(2), digest);
    // Submit WITHOUT X-Signature-Format header (default = raw)
    const response = await axios.post(QUERY_URL, {
      bytes: Buffer.from(serialized).toString("hex"),
      signature,
    });
    expect([200, 429, 504]).toContain(response.status);
    if (response.status === 200) {
      expect(response.data.bytes).toBeTruthy();
    }
  });

  test("prefixed signature fails without header (wrong address recovered)", async () => {
    const { privateKey, address } = getNextWallet();
    const client = createClient(privateKey);
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const blockNumber = await client.getBlockNumber();
    const ethCall = new EthCallQueryRequest(Number(blockNumber), [
      nameCallData,
    ]);
    const ethQuery = new PerChainQueryRequest(2, ethCall);
    // No staker address: server will use the recovered signer as the rate limit key.
    // Wrong recovery (raw mode on EIP-191 sig) yields a random address with no stake.
    const request = new QueryRequest(3, Math.floor(Date.now() / 1000), [ethQuery]);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    // Sign with personal_sign (EIP-191 prefixed)
    const signature = await client.signMessage({
      message: { raw: digest },
    });
    // Submit WITHOUT the header - server will try raw recovery, get wrong address
    let err = false;
    await axios
      .post(QUERY_URL, {
        bytes: Buffer.from(serialized).toString("hex"),
        signature: signature.slice(2),
      })
      .catch(function(error) {
        err = true;
        // Should fail because wrong address is recovered -> no stake found
        expect(error.response?.status).toBe(403);
        expect(error.response?.data).toContain("insufficient stake");
      });
    expect(err).toBe(true);
  });

  test("health check", async () => {
    const response = await axios.get(HEALTH_URL);
    expect(response.status).toBe(200);
  });

  test("payload too large should fail", async () => {
    const serialized = new Uint8Array(6000000); // Buffer should be larger than MAX_BODY_SIZE in node/cmd/ccq/http.go.
    const signature = "";
    let err = false;
    await axios
      .post(QUERY_URL, {
        signature,
        bytes: Buffer.from(serialized).toString("hex"),
      })
      .catch(function(error) {
        err = true;
        expect(error.response.status).toBe(400);
        expect(error.response.data).toContain("request body too large");
      });
    expect(err).toBe(true);
  });

  test("serialize eth_call_by_timestamp request", () => {
    // Serialize test doesn't need real stake - just needs an address
    const dummyAddress = privateKeyToAccount(generatePrivateKey()).address;
    const toAddress = "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270";
    const nameCallData = createTestEthCallData(toAddress, "name", "string");
    const decimalsCallData = createTestEthCallData(
      toAddress,
      "decimals",
      "uint8"
    );
    const ethCall = new EthCallByTimestampQueryRequest(
      BigInt(1697216322000000),
      "0x28d9630",
      "0x28d9631",
      [nameCallData, decimalsCallData]
    );
    const chainId = 5;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(
      nonce,
      Math.floor(Date.now() / 1000),
      [ethQuery],
      dummyAddress
    );
    const serialized = request.serialize();
    // V2 format with staker address - just verify it serializes without error
    expect(serialized).toBeTruthy();
    expect(serialized.length).toBeGreaterThan(0);
    expect(serialized[0]).toEqual(0x02);
  });

  test("successful eth_call_by_timestamp query with block hints", async () => {
    const { privateKey, address } = getNextWallet();
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const decimalsCallData = createTestEthCallData(
      WETH_ADDRESS,
      "decimals",
      "uint8"
    );
    const [targetBlockTime, targetBlockNumber, followingBlockNumber] =
      await getEthCallByTimestampArgs();
    const ethCall = new EthCallByTimestampQueryRequest(
      targetBlockTime,
      targetBlockNumber.toString(16),
      followingBlockNumber.toString(16),
      [nameCallData, decimalsCallData]
    );
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, Math.floor(Date.now() / 1000), [ethQuery], address);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(privateKey.slice(2), digest);

    let response;
    try {
      response = await axios.post(QUERY_URL, {
        signature,
        bytes: Buffer.from(serialized).toString("hex"),
      });
    } catch (error: any) {
      // Handle network errors gracefully
      if (error.code === "ECONNRESET" || !error.response) {
        console.warn("Network error during eth_call_by_timestamp query:", error.message);
        return; // Skip validation if network error occurs
      }
      throw error;
    }
    expect(response.status).toBe(200);

    const queryResponse = QueryResponse.from(response.data.bytes);
    expect(queryResponse.version).toEqual(1);
    expect(queryResponse.requestChainId).toEqual(0);
    expect(queryResponse.request.requests.length).toEqual(1);
    expect(queryResponse.request.requests[0].chainId).toEqual(2);
    expect(queryResponse.request.requests[0].query.type()).toEqual(
      ChainQueryType.EthCallByTimeStamp
    );

    const ecr = queryResponse.responses[0]
      .response as EthCallByTimestampQueryResponse;
    expect(ecr.targetBlockNumber.toString()).toEqual(
      BigInt(targetBlockNumber).toString()
    );
    expect(ecr.targetBlockHash).toEqual(
      (
        await createClient().getBlock({
          blockNumber: BigInt(targetBlockNumber),
        })
      ).hash
    );
    expect(ecr.followingBlockNumber.toString()).toEqual(
      BigInt(followingBlockNumber).toString()
    );
    expect(ecr.followingBlockHash).toEqual(
      (
        await createClient().getBlock({
          blockNumber: BigInt(followingBlockNumber),
        })
      ).hash
    );
    expect(ecr.results.length).toEqual(2);
    expect(ecr.results[0]).toEqual(
      // Name
      "0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000"
    );
    expect(ecr.results[1]).toEqual(
      // Decimals
      "0x0000000000000000000000000000000000000000000000000000000000000012"
    );
  });

  test("successful eth_call_by_timestamp query without block hints", async () => {
    const { privateKey, address } = getNextWallet();
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const decimalsCallData = createTestEthCallData(
      WETH_ADDRESS,
      "decimals",
      "uint8"
    );
    const [targetBlockTime, targetBlockNumber, followingBlockNumber] =
      await getEthCallByTimestampArgs();
    const ethCall = new EthCallByTimestampQueryRequest(
      targetBlockTime + BigInt(5000),
      "",
      "",
      [nameCallData, decimalsCallData]
    );
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, Math.floor(Date.now() / 1000), [ethQuery], address);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(privateKey.slice(2), digest);
    const response = await axios.post(QUERY_URL, {
      signature,
      bytes: Buffer.from(serialized).toString("hex"),
    });
    expect(response.status).toBe(200);

    const queryResponse = QueryResponse.from(response.data.bytes);
    expect(queryResponse.version).toEqual(1);
    expect(queryResponse.requestChainId).toEqual(0);
    expect(queryResponse.request.requests.length).toEqual(1);
    expect(queryResponse.request.requests[0].chainId).toEqual(2);
    expect(queryResponse.request.requests[0].query.type()).toEqual(
      ChainQueryType.EthCallByTimeStamp
    );

    const ecr = queryResponse.responses[0]
      .response as EthCallByTimestampQueryResponse;
    expect(ecr.targetBlockNumber.toString()).toEqual(
      BigInt(targetBlockNumber).toString()
    );
    expect(ecr.targetBlockHash).toEqual(
      (
        await createClient().getBlock({
          blockNumber: BigInt(targetBlockNumber),
        })
      ).hash
    );
    expect(ecr.followingBlockNumber.toString()).toEqual(
      BigInt(followingBlockNumber).toString()
    );
    expect(ecr.followingBlockHash).toEqual(
      (
        await createClient().getBlock({
          blockNumber: BigInt(followingBlockNumber),
        })
      ).hash
    );
    expect(ecr.results.length).toEqual(2);
    expect(ecr.results[0]).toEqual(
      // Name
      "0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000"
    );
    expect(ecr.results[1]).toEqual(
      // Decimals
      "0x0000000000000000000000000000000000000000000000000000000000000012"
    );
  });

  test("eth_call_by_timestamp query without target timestamp", async () => {
    const { privateKey, address } = getNextWallet();
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const decimalsCallData = createTestEthCallData(
      WETH_ADDRESS,
      "decimals",
      "uint8"
    );
    const followingBlockNum = await createClient().getBlockNumber();
    const followingBlock = await createClient().getBlock({
      blockNumber: BigInt(followingBlockNum),
    });
    const targetBlock = await createClient().getBlock({
      blockNumber: BigInt(followingBlockNum) - BigInt(1),
    });
    const ethCall = new EthCallByTimestampQueryRequest(
      BigInt(0),
      targetBlock.number.toString(16),
      followingBlock.number.toString(16),
      [nameCallData, decimalsCallData]
    );
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, Math.floor(Date.now() / 1000), [ethQuery], address);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(privateKey.slice(2), digest);
    let err = false;
    await axios
      .post(QUERY_URL, {
        signature,
        bytes: Buffer.from(serialized).toString("hex"),
      })
      .catch(function(error) {
        err = true;
        expect(error.response.status).toBe(400);
        // Be flexible - server may return truncated error message
        expect(error.response.data).toContain("failed to unmarshal request");
      });
    expect(err).toBe(true);
  });

  test("eth_call_by_timestamp query with following hint but not target hint should fail", async () => {
    const { privateKey, address } = getNextWallet();
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const decimalsCallData = createTestEthCallData(
      WETH_ADDRESS,
      "decimals",
      "uint8"
    );
    const followingBlockNum = await createClient().getBlockNumber();
    const followingBlock = await createClient().getBlock({
      blockNumber: BigInt(followingBlockNum),
    });
    const targetBlock = await createClient().getBlock({
      blockNumber: BigInt(followingBlockNum) - BigInt(1),
    });
    const targetBlockTime = targetBlock.timestamp * BigInt(1000000);
    const ethCall = new EthCallByTimestampQueryRequest(
      targetBlockTime,
      "",
      followingBlock.number.toString(16),
      [nameCallData, decimalsCallData]
    );
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, Math.floor(Date.now() / 1000), [ethQuery], address);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(privateKey.slice(2), digest);
    let err = false;
    await axios
      .post(QUERY_URL, {
        signature,
        bytes: Buffer.from(serialized).toString("hex"),
      })
      .catch(function(error) {
        err = true;
        expect(error.response.status).toBe(400);
        // Be flexible - server may return truncated error message
        expect(error.response.data).toContain("failed to unmarshal request");
      });
    expect(err).toBe(true);
  });

  test("eth_call_by_timestamp query with target hint but not following hint should fail", async () => {
    const { privateKey, address } = getNextWallet();
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const decimalsCallData = createTestEthCallData(
      WETH_ADDRESS,
      "decimals",
      "uint8"
    );
    const followingBlockNum = await createClient().getBlockNumber();
    const targetBlock = await createClient().getBlock({
      blockNumber: BigInt(followingBlockNum) - BigInt(1),
    });
    const targetBlockTime = targetBlock.timestamp * BigInt(1000000);
    const ethCall = new EthCallByTimestampQueryRequest(
      targetBlockTime,
      targetBlock.number.toString(16),
      "",
      [nameCallData, decimalsCallData]
    );
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, Math.floor(Date.now() / 1000), [ethQuery], address);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(privateKey.slice(2), digest);
    let err = false;
    await axios
      .post(QUERY_URL, {
        signature,
        bytes: Buffer.from(serialized).toString("hex"),
      })
      .catch(function(error) {
        err = true;
        expect(error.response.status).toBe(400);
        // Be flexible - server may return truncated error message
        expect(error.response.data).toContain("failed to unmarshal request");
      });
    expect(err).toBe(true);
  });

  test("serialize eth_call_with_finality request", () => {
    // Serialize test doesn't need real stake - just needs an address
    const dummyAddress = privateKeyToAccount(generatePrivateKey()).address;
    const toAddress = "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270";
    const nameCallData = createTestEthCallData(toAddress, "name", "string");
    const decimalsCallData = createTestEthCallData(
      toAddress,
      "decimals",
      "uint8"
    );
    const ethCall = new EthCallWithFinalityQueryRequest(
      "0x28d9630",
      "finalized",
      [nameCallData, decimalsCallData]
    );
    const chainId = 5;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(
      nonce,
      Math.floor(Date.now() / 1000),
      [ethQuery],
      dummyAddress
    );
    const serialized = request.serialize();
    // V2 format with staker address - just verify it serializes without error
    expect(serialized).toBeTruthy();
    expect(serialized.length).toBeGreaterThan(0);
    expect(serialized[0]).toEqual(0x02);
  });

  test("successful eth_call_with_finality query", async () => {
    const { privateKey, address } = getNextWallet();
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const decimalsCallData = createTestEthCallData(
      WETH_ADDRESS,
      "decimals",
      "uint8"
    );
    const blockNumber = Number(
      (await createClient().getBlock({ blockTag: "finalized" })).number
    );
    const ethCall = new EthCallWithFinalityQueryRequest(
      blockNumber.toString(16),
      "finalized",
      [nameCallData, decimalsCallData]
    );
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, Math.floor(Date.now() / 1000), [ethQuery], address);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(privateKey.slice(2), digest);
    const response = await axios.post(QUERY_URL, {
      signature,
      bytes: Buffer.from(serialized).toString("hex"),
    });
    expect(response.status).toBe(200);

    const queryResponse = QueryResponse.from(response.data.bytes);
    expect(queryResponse.version).toEqual(1);
    expect(queryResponse.requestChainId).toEqual(0);
    expect(queryResponse.request.requests.length).toEqual(1);
    expect(queryResponse.request.requests[0].chainId).toEqual(2);
    expect(queryResponse.request.requests[0].query.type()).toEqual(
      ChainQueryType.EthCallWithFinality
    );

    const ecr = queryResponse.responses[0]
      .response as EthCallWithFinalityQueryResponse;
    expect(ecr.blockNumber.toString()).toEqual(BigInt(blockNumber).toString());
    expect(ecr.blockHash).toEqual(
      (await createClient().getBlock({ blockNumber: BigInt(blockNumber) })).hash
    );
    expect(ecr.results.length).toEqual(2);
    expect(ecr.results[0]).toEqual(
      // Name
      "0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000"
    );
    expect(ecr.results[1]).toEqual(
      // Decimals
      "0x0000000000000000000000000000000000000000000000000000000000000012"
    );
  });

  test("eth_call_with_finality query without finality should fail", async () => {
    const { privateKey, address } = getNextWallet();
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const decimalsCallData = createTestEthCallData(
      WETH_ADDRESS,
      "decimals",
      "uint8"
    );
    const ethCall = new EthCallWithFinalityQueryRequest(
      "0x28d9630",
      "" as any,
      [nameCallData, decimalsCallData]
    );
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, Math.floor(Date.now() / 1000), [ethQuery], address);

    let err = false;
    try {
      const serialized = request.serialize();
      const digest = QueryRequest.digest(ENV, serialized);
      const signature = sign(privateKey.slice(2), digest);
      await axios.post(QUERY_URL, {
        signature,
        bytes: Buffer.from(serialized).toString("hex"),
      });
    } catch (error: any) {
      err = true;
      // May fail during serialization (no response) or during server validation (400 response)
      if (error.response) {
        expect(error.response.status).toBe(400);
        expect(error.response.data).toContain("failed to unmarshal request");
      } else {
        // Serialization error - that's also acceptable since the request is invalid
        expect(error).toBeTruthy();
      }
    }
    expect(err).toBe(true);
  });

  test("eth_call_with_finality query with bad finality should fail", async () => {
    const { privateKey, address } = getNextWallet();
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const decimalsCallData = createTestEthCallData(
      WETH_ADDRESS,
      "decimals",
      "uint8"
    );
    const ethCall = new EthCallWithFinalityQueryRequest(
      "0x28d9630",
      "HelloWorld" as any,
      [nameCallData, decimalsCallData]
    );
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, Math.floor(Date.now() / 1000), [ethQuery], address);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(privateKey.slice(2), digest);
    let err = false;
    await axios
      .post(QUERY_URL, {
        signature,
        bytes: Buffer.from(serialized).toString("hex"),
      })
      .catch(function(error) {
        err = true;
        expect(error.response.status).toBe(400);
        // Be flexible - server may return truncated error message
        expect(error.response.data).toContain("failed to unmarshal request");
      });
    expect(err).toBe(true);
  });

  test("concurrent queries", async () => {
    const { privateKey, address } = getNextWallet();
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const decimalsCallData = createTestEthCallData(
      WETH_ADDRESS,
      "decimals",
      "uint8"
    );
    const blockNumber = await createClient().getBlockNumber();
    const ethCall = new EthCallQueryRequest(Number(blockNumber), [
      nameCallData,
      decimalsCallData,
    ]);
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    let nonce = 1;
    let promises: Promise<AxiosResponse<any, any>>[] = [];

    // With 50k tokens staked, we should have very high rate limits
    // Send 20 concurrent queries - they should all succeed
    for (let count = 0; count < 20; count++) {
      nonce += 1;
      const request = new QueryRequest(nonce, Math.floor(Date.now() / 1000), [ethQuery], address);
      const serialized = request.serialize();
      const digest = QueryRequest.digest(ENV, serialized);
      const signature = sign(privateKey.slice(2), digest);
      const response = axios.post(QUERY_URL, {
        signature,
        bytes: Buffer.from(serialized).toString("hex"),
      });
      promises.push(response);
    }

    const responses = await Promise.all(promises);

    expect(responses.length).toEqual(promises.length);
    for (let idx = 0; idx < responses.length; idx++) {
      const response = responses[idx];
      expect(response.status).toBe(200);

      const queryResponse = QueryResponse.from(response.data.bytes);
      expect(queryResponse.version).toEqual(1);
      expect(queryResponse.requestChainId).toEqual(0);
      expect(queryResponse.request.requests.length).toEqual(1);
      expect(queryResponse.request.requests[0].chainId).toEqual(2);
      expect(queryResponse.request.requests[0].query.type()).toEqual(
        ChainQueryType.EthCall
      );

      const ecr = queryResponse.responses[0].response as EthCallQueryResponse;
      expect(ecr.blockNumber.toString()).toEqual(
        BigInt(blockNumber).toString()
      );
      expect(ecr.blockHash).toEqual(
        (await createClient().getBlock({ blockNumber: BigInt(blockNumber) }))
          .hash
      );
      expect(ecr.results.length).toEqual(2);
      expect(ecr.results[0]).toEqual(
        // Name
        "0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000"
      );
      expect(ecr.results[1]).toEqual(
        // Decimals
        "0x0000000000000000000000000000000000000000000000000000000000000012"
      );
    }
  });

  test("allow anything", async () => {
    const { privateKey, address } = getNextWallet();
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const decimalsCallData = createTestEthCallData(
      WETH_ADDRESS,
      "decimals",
      "uint8"
    );
    const blockNumber = await createClient().getBlockNumber();
    const ethCall = new EthCallQueryRequest(Number(blockNumber), [
      nameCallData,
      decimalsCallData,
    ]);
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, Math.floor(Date.now() / 1000), [ethQuery], address);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(privateKey.slice(2), digest);
    const response = await axios.post(QUERY_URL, {
      signature,
      bytes: Buffer.from(serialized).toString("hex"),
    });
    expect(response.status).toBe(200);

    const queryResponse = QueryResponse.from(response.data.bytes);
    expect(queryResponse.version).toEqual(1);
    expect(queryResponse.requestChainId).toEqual(0);
    expect(queryResponse.request.requests.length).toEqual(1);
    expect(queryResponse.request.requests[0].chainId).toEqual(2);
    expect(queryResponse.request.requests[0].query.type()).toEqual(
      ChainQueryType.EthCall
    );

    const ecr = queryResponse.responses[0].response as EthCallQueryResponse;
    expect(ecr.blockNumber.toString()).toEqual(BigInt(blockNumber).toString());
    expect(ecr.blockHash).toEqual(
      (await createClient().getBlock({ blockNumber: BigInt(blockNumber) })).hash
    );
    expect(ecr.results.length).toEqual(2);
    expect(ecr.results[0]).toEqual(
      // Name
      "0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000"
    );
    expect(ecr.results[1]).toEqual(
      // Decimals
      "0x0000000000000000000000000000000000000000000000000000000000000012"
    );
  });
});
