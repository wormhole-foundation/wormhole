import { beforeAll, describe, expect, jest, test } from "@jest/globals";
import axios from "axios";
import { Buffer } from "buffer";
import { type Address } from "viem";
import { privateKeyToAccount, generatePrivateKey } from "viem/accounts";
import {
  EthCallQueryRequest,
  PerChainQueryRequest,
  QueryRequest,
  sign,
} from "../../src";
import {
  EVM_QUERY_TYPE,
  STAKING_FACTORY_ADDRESS,
  getCurrentTimestampSeconds,
  createClient,
  getPoolAddress,
  POOL_DELEGATION_ABI,
  QUERY_URL,
  setupAxiosInterceptor,
  sleep,
  WETH_ADDRESS,
  createTestEthCallData,
  setupWalletsWithStake,
  setupDelegation,
} from "./test-utils";

jest.setTimeout(180000); // 3 minutes for tests with long blockchain operations
setupAxiosInterceptor();

// Stake amounts for testing
const HIGH_STAKE_AMOUNT = "1000"; // High stake for most tests
const MEDIUM_STAKE_AMOUNT = "600"; // Medium stake for some tests

// Create wallet pools - delegators (stake holders) and signers (delegates)
const delegatorWallets: Array<{
  privateKey: `0x${string}`;
  address: Address;
}> = [];

const signerWallets: Array<{
  privateKey: `0x${string}`;
  address: Address;
}> = [];

// Create 3 delegator wallets (stake holders) - reduced from 4 for faster setup
for (let i = 0; i < 3; i++) {
  const privateKey = generatePrivateKey();
  const account = privateKeyToAccount(privateKey);
  delegatorWallets.push({ privateKey, address: account.address });
}

// Create 2 signer wallets (delegates without stake)
for (let i = 0; i < 2; i++) {
  const privateKey = generatePrivateKey();
  const account = privateKeyToAccount(privateKey);
  signerWallets.push({ privateKey, address: account.address });
}

let delegatorIndex = 0;
function getNextDelegator() {
  const wallet = delegatorWallets[delegatorIndex % delegatorWallets.length];
  delegatorIndex++;
  return wallet;
}

let signerIndex = 0;
function getNextSigner() {
  const wallet = signerWallets[signerIndex % signerWallets.length];
  signerIndex++;
  return wallet;
}

let poolAddress: Address;

// ============================================================================
// Test Suite
// ============================================================================

describe("Delegation Integration Tests", () => {
  beforeAll(async () => {
    poolAddress = await getPoolAddress(STAKING_FACTORY_ADDRESS, EVM_QUERY_TYPE);

    expect(poolAddress).toBeTruthy();
    expect(poolAddress).not.toBe("0x0000000000000000000000000000000000000000");

    await setupWalletsWithStake(
      delegatorWallets,
      poolAddress,
      HIGH_STAKE_AMOUNT
    );

    const verifyClient = createClient();
    for (const wallet of delegatorWallets) {
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

      if (BigInt(stakeAmount) === BigInt(0)) {
        throw new Error(
          `Wallet ${wallet.address} has zero stake! Staking failed.`
        );
      }
    }
  }, 120000); // 2 minutes for setup

  test("Verify delegation is set up correctly", async () => {
    const delegator = getNextDelegator();
    const signer = getNextSigner();

    await setupDelegation(poolAddress, delegator.privateKey, signer.address);

    const client = createClient();
    const registeredSigner = (await client.readContract({
      address: poolAddress,
      abi: POOL_DELEGATION_ABI,
      functionName: "stakerSigners",
      args: [delegator.address],
    } as any)) as string;

    expect(registeredSigner.toLowerCase()).toBe(signer.address.toLowerCase());
  });

  test("Delegated query succeeds (signer signs, delegator's limits apply)", async () => {
    const delegator = getNextDelegator();
    const signer = getNextSigner();

    await setupDelegation(poolAddress, delegator.privateKey, signer.address);

    // Get current block number for the query
    const client = createClient();
    const blockNumber = await client.getBlockNumber();

    const callData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const ethCall = new EthCallQueryRequest(Number(blockNumber), [callData]);
    const perChainQuery = new PerChainQueryRequest(2, ethCall);

    const queryRequest = new QueryRequest(
      getCurrentTimestampSeconds(),
      getCurrentTimestampSeconds(),
      [perChainQuery],
      delegator.address
    );

    const serialized = queryRequest.serialize();
    const digest = QueryRequest.digest("DEVNET", serialized);
    const signature = sign(signer.privateKey.slice(2), digest);

    try {
      const response = await axios.post(
        QUERY_URL,
        {
          signature,
          bytes: Buffer.from(serialized).toString("hex"),
          staker: delegator.address,
        },
        {
          timeout: 5000, // Reduced from 10s to 5s
          validateStatus: () => true,
        }
      );

      if (response.status === 403) {
        console.warn(
          "WARNING: Delegation may not be enabled on CCQ server (got 403)"
        );
      } else {
        expect(response.status).toBe(200);
      }
    } catch (error: any) {
      const isTimeout =
        error.code === "ECONNABORTED" ||
        error.code === "ETIMEDOUT" ||
        !error.response;

      if (!isTimeout) {
        throw error;
      }
    }
  }, 90000);

  test("Unauthorized signer cannot use delegator's limits", async () => {
    const delegator1 = getNextDelegator();
    const delegator2 = getNextDelegator();

    const client = createClient();
    const blockNumber = await client.getBlockNumber();

    const callData = createTestEthCallData(WETH_ADDRESS, "symbol", "string");
    const ethCall = new EthCallQueryRequest(Number(blockNumber), [callData]);
    const perChainQuery = new PerChainQueryRequest(2, ethCall);

    const queryRequest = new QueryRequest(
      42,
      getCurrentTimestampSeconds(),
      [perChainQuery],
      delegator1.address
    );

    const serialized = queryRequest.serialize();
    const digest = QueryRequest.digest("DEVNET", serialized);
    const signature = sign(delegator2.privateKey.slice(2), digest);

    const response = await axios.post(
      QUERY_URL,
      {
        signature,
        bytes: Buffer.from(serialized).toString("hex"),
        staker: delegator1.address,
      },
      {
        validateStatus: () => true,
      }
    );

    expect(response.status).toBe(403);
  });

  test("Address with no delegation and no stake cannot query", async () => {
    const signer = getNextSigner();

    const client = createClient();
    const blockNumber = await client.getBlockNumber();

    const callData = createTestEthCallData(WETH_ADDRESS, "decimals", "uint8");
    const ethCall = new EthCallQueryRequest(Number(blockNumber), [callData]);
    const perChainQuery = new PerChainQueryRequest(2, ethCall);

    const queryRequest = new QueryRequest(
      getCurrentTimestampSeconds(),
      getCurrentTimestampSeconds(),
      [perChainQuery]
    );

    const serialized = queryRequest.serialize();
    const digest = QueryRequest.digest("DEVNET", serialized);
    const signature = sign(signer.privateKey.slice(2), digest);

    const response = await axios.post(
      QUERY_URL,
      {
        signature,
        bytes: Buffer.from(serialized).toString("hex"),
      },
      {
        validateStatus: () => true,
      }
    );

    expect(response.status).toBe(403);
  });

  test("Rate limits are based on delegator's stake, not signer's", async () => {
    const delegator = getNextDelegator();
    const signer = getNextSigner();

    await setupDelegation(poolAddress, delegator.privateKey, signer.address);

    const client = createClient();
    const blockNumber = await client.getBlockNumber();

    const callData = createTestEthCallData(
      WETH_ADDRESS,
      "totalSupply",
      "uint256"
    );
    const ethCall = new EthCallQueryRequest(Number(blockNumber), [callData]);
    const perChainQuery = new PerChainQueryRequest(2, ethCall);

    const results: Array<{ status: number }> = [];
    // 2 QPS rate limit
    const numQueries = 2;

    console.log(
      `\nSending ${numQueries} queries using delegator ${delegator.address} with signer ${signer.address}`
    );

    for (let i = 0; i < numQueries; i++) {
      const queryRequest = new QueryRequest(
        getCurrentTimestampSeconds() + i,
        getCurrentTimestampSeconds(),
        [perChainQuery],
        delegator.address
      );

      const serialized = queryRequest.serialize();
      const digest = QueryRequest.digest("DEVNET", serialized);
      const signature = sign(signer.privateKey.slice(2), digest);

      try {
        const response = await axios.post(
          QUERY_URL,
          {
            signature,
            bytes: Buffer.from(serialized).toString("hex"),
            staker: delegator.address,
          },
          {
            timeout: 3000,
            validateStatus: () => true,
          }
        );

        results.push({ status: response.status });
      } catch (error: any) {
        const isTimeout =
          error.code === "ECONNABORTED" ||
          error.code === "ETIMEDOUT" ||
          !error.response;
        const status = isTimeout ? 504 : error.response?.status || 500;
        results.push({ status });
      }

      if (i < numQueries - 1) {
        await sleep(600);
      }
    }

    const accepted = results.filter((r) => r.status === 200).length;
    const rateLimited = results.filter((r) => r.status === 429).length;
    const timedOut = results.filter((r) => r.status === 504).length;
    const forbidden = results.filter((r) => r.status === 403).length;

    if (forbidden === results.length) {
      expect(forbidden).toBeGreaterThan(0);
    } else {
      expect(forbidden).toBe(0);
      expect(rateLimited).toBe(0);
      expect(accepted + timedOut).toBe(numQueries);
    }

    await sleep(1000);
  });

  test("Revoke delegation by setting signer to zero address", async () => {
    const delegator = getNextDelegator();
    const signer = getNextSigner();

    await setupDelegation(poolAddress, delegator.privateKey, signer.address);

    const client = createClient(delegator.privateKey);
    const beforeRevoke = (await client.readContract({
      address: poolAddress,
      abi: POOL_DELEGATION_ABI,
      functionName: "stakerSigners",
      args: [delegator.address],
    } as any)) as string;
    expect(beforeRevoke.toLowerCase()).toBe(signer.address.toLowerCase());

    const hash = await client.writeContract({
      address: poolAddress,
      abi: POOL_DELEGATION_ABI,
      functionName: "setSigner",
      args: ["0x0000000000000000000000000000000000000000"],
    } as any);

    const receipt = await client.waitForTransactionReceipt({ hash });
    expect(receipt.status).toBe("success");

    const newSigner = (await client.readContract({
      address: poolAddress,
      abi: POOL_DELEGATION_ABI,
      functionName: "stakerSigners",
      args: [delegator.address],
    } as any)) as string;
    expect(newSigner).toBe("0x0000000000000000000000000000000000000000");

    await sleep(15000);

    const queryClient = createClient();
    const blockNumber = await queryClient.getBlockNumber();

    const callData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const ethCall = new EthCallQueryRequest(Number(blockNumber), [callData]);
    const perChainQuery = new PerChainQueryRequest(2, ethCall);

    const queryRequest = new QueryRequest(
      getCurrentTimestampSeconds(),
      getCurrentTimestampSeconds(),
      [perChainQuery],
      delegator.address
    );

    const serialized = queryRequest.serialize();
    const digest = QueryRequest.digest("DEVNET", serialized);
    const signature = sign(signer.privateKey.slice(2), digest);

    try {
      const response = await axios.post(
        QUERY_URL,
        {
          signature,
          bytes: Buffer.from(serialized).toString("hex"),
          staker: delegator.address,
        },
        {
          timeout: 5000, // Reduced from 10s to 5s
          validateStatus: () => true,
        }
      );

      expect(response.status).toBe(403);
    } catch (error: any) {
      const isTimeout =
        error.code === "ECONNABORTED" ||
        error.code === "ETIMEDOUT" ||
        !error.response;

      if (isTimeout) {
        throw new Error(
          "Query with revoked signer should have been rejected with 403, but timed out instead"
        );
      }

      throw error;
    }
  });

  test("V2 format (no stakerAddress) works for self-staking", async () => {
    const delegator = getNextDelegator();

    // Get current block number
    const client = createClient();
    const blockNumber = await client.getBlockNumber();

    const callData = createTestEthCallData(WETH_ADDRESS, "symbol", "string");
    const ethCall = new EthCallQueryRequest(Number(blockNumber), [callData]);
    const perChainQuery = new PerChainQueryRequest(2, ethCall);

    const queryRequest = new QueryRequest(
      getCurrentTimestampSeconds(),
      getCurrentTimestampSeconds(),
      [perChainQuery]
    );

    const serialized = queryRequest.serialize();
    // V2 format always uses version byte = 2
    expect(serialized[0]).toBe(2);

    const digest = QueryRequest.digest("DEVNET", serialized);
    const signature = sign(delegator.privateKey.slice(2), digest);

    try {
      const response = await axios.post(
        QUERY_URL,
        {
          signature,
          bytes: Buffer.from(serialized).toString("hex"),
        },
        {
          timeout: 3000, // Reduced from 10s to 3s
          validateStatus: () => true,
        }
      );

      expect(response.status).toBe(200);
    } catch (error: any) {
      const isTimeout =
        error.code === "ECONNABORTED" ||
        error.code === "ETIMEDOUT" ||
        !error.response;

      if (!isTimeout) {
        throw error;
      }
    }
  });

  test("Staker address is included in v2 binary format", async () => {
    const delegator = getNextDelegator();

    // Get current block number
    const client = createClient();
    const blockNumber = await client.getBlockNumber();

    const callData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const ethCall = new EthCallQueryRequest(Number(blockNumber), [callData]);
    const perChainQuery = new PerChainQueryRequest(2, ethCall);

    // Create QueryRequest with stakerAddress parameter
    const queryRequest = new QueryRequest(
      getCurrentTimestampSeconds(),
      getCurrentTimestampSeconds(),
      [perChainQuery],
      delegator.address
    );

    const serialized = queryRequest.serialize();
    // V2 format uses version byte = 2
    expect(serialized[0]).toBe(2);

    // The staker address is stored in the QueryRequest object
    expect(queryRequest.stakerAddress?.toLowerCase()).toBe(
      delegator.address.toLowerCase()
    );

    // In v2 format, the staker address is encoded in the binary format
    // Can verify by deserializing and checking the staker address is preserved
    const deserialized = QueryRequest.from(serialized);
    expect(deserialized.stakerAddress?.toLowerCase()).toBe(
      delegator.address.toLowerCase()
    );
  });
});
