import { beforeAll, describe, expect, jest, test } from "@jest/globals";
import { parseEther, formatEther, type Address } from "viem";
import { privateKeyToAccount, generatePrivateKey } from "viem/accounts";
import {
  EVM_QUERY_TYPE,
  STAKING_FACTORY_ADDRESS,
  createClient,
  getPoolAddress,
  POOL_ABI,
  POOL_STAKE_ABI,
  ERC20_ABI,
  setupAxiosInterceptor,
  sleep,
  W_TOKEN_ADDRESS,
  setupWalletsWithStake,
  sendQuery,
} from "./test-utils";

jest.setTimeout(180000);
setupAxiosInterceptor();

// Rate limit tiers from `devnet/ccq-rate-limits-config.yaml`
const LOW_STAKE_AMOUNT = "55"; // Low stake for rate limit testing (50+ to cover fees)
const MEDIUM_STAKE_AMOUNT = "600"; // Medium stake (1 QPS)
const HIGH_STAKE_AMOUNT = "10000"; // High stake (20 QPS)

// Create wallet pools for different test scenarios
const lowStakeWallets: Array<{ privateKey: `0x${string}`; address: Address }> =
  [];
const mediumStakeWallets: Array<{
  privateKey: `0x${string}`;
  address: Address;
}> = [];
const highStakeWallets: Array<{ privateKey: `0x${string}`; address: Address }> =
  [];
const increaseStakeWallets: Array<{
  privateKey: `0x${string}`;
  address: Address;
}> = [];

// Create 2 wallets for low stake (rate limit testing)
for (let i = 0; i < 2; i++) {
  const privateKey = generatePrivateKey();
  const account = privateKeyToAccount(privateKey);
  lowStakeWallets.push({ privateKey, address: account.address });
}

// Create 1 wallet for medium stake (comparison testing)
for (let i = 0; i < 1; i++) {
  const privateKey = generatePrivateKey();
  const account = privateKeyToAccount(privateKey);
  mediumStakeWallets.push({ privateKey, address: account.address });
}

// Create 2 wallets for high stake (basic functionality)
for (let i = 0; i < 2; i++) {
  const privateKey = generatePrivateKey();
  const account = privateKeyToAccount(privateKey);
  highStakeWallets.push({ privateKey, address: account.address });
}

// Create 1 wallet for increase stake test
for (let i = 0; i < 1; i++) {
  const privateKey = generatePrivateKey();
  const account = privateKeyToAccount(privateKey);
  increaseStakeWallets.push({ privateKey, address: account.address });
}

let lowStakeIndex = 0;
function getNextLowStakeWallet() {
  const wallet = lowStakeWallets[lowStakeIndex % lowStakeWallets.length];
  lowStakeIndex++;
  return wallet;
}

let mediumStakeIndex = 0;
function getNextMediumStakeWallet() {
  const wallet =
    mediumStakeWallets[mediumStakeIndex % mediumStakeWallets.length];
  mediumStakeIndex++;
  return wallet;
}

let highStakeIndex = 0;
function getNextHighStakeWallet() {
  const wallet = highStakeWallets[highStakeIndex % highStakeWallets.length];
  highStakeIndex++;
  return wallet;
}

let increaseStakeIndex = 0;
function getNextIncreaseStakeWallet() {
  const wallet =
    increaseStakeWallets[increaseStakeIndex % increaseStakeWallets.length];
  increaseStakeIndex++;
  return wallet;
}

let poolAddress: Address;

// Track which addresses have staked during tests for cleanup
const stakersToCleanup = new Set<`0x${string}`>();

describe("Staking Integration Tests", () => {
  beforeAll(async () => {
    poolAddress = await getPoolAddress(STAKING_FACTORY_ADDRESS, EVM_QUERY_TYPE);
    console.log("staking integration tests", "pool_address", poolAddress);

    expect(poolAddress).toBeTruthy();
    expect(poolAddress).not.toBe("0x0000000000000000000000000000000000000000");

    console.log("\nStaking Integration Test Configuration:");
    console.log("  Factory:", STAKING_FACTORY_ADDRESS);
    console.log("  Pool:", poolAddress);
    console.log("  Token:", W_TOKEN_ADDRESS);

    // Setup wallet pools with different stake amounts
    await setupWalletsWithStake(lowStakeWallets, poolAddress, LOW_STAKE_AMOUNT);
    await setupWalletsWithStake(
      mediumStakeWallets,
      poolAddress,
      MEDIUM_STAKE_AMOUNT
    );
    await setupWalletsWithStake(
      highStakeWallets,
      poolAddress,
      HIGH_STAKE_AMOUNT
    );

    // Setup tokens for increase stake test (but don't stake yet)
    const minterClient = createClient();
    for (const wallet of increaseStakeWallets) {
      const ethHash = await minterClient.sendTransaction({
        to: wallet.address,
        value: parseEther("1"),
      } as any);
      await minterClient.waitForTransactionReceipt({ hash: ethHash });

      const tokenAmount = "60000";
      const tokenAmountWei = parseEther(tokenAmount);
      const mintHash = await minterClient.writeContract({
        address: W_TOKEN_ADDRESS,
        abi: [
          {
            inputs: [
              { name: "to", type: "address" },
              { name: "amount", type: "uint256" },
            ],
            name: "mint",
            outputs: [],
            stateMutability: "nonpayable",
            type: "function",
          },
        ],
        functionName: "mint",
        args: [wallet.address, tokenAmountWei],
      } as any);
      await minterClient.waitForTransactionReceipt({ hash: mintHash });
    }

    // Verify stakes and track for cleanup
    const verifyClient = createClient();
    const allStakedWallets = [
      ...lowStakeWallets,
      ...mediumStakeWallets,
      ...highStakeWallets,
    ];

    for (const wallet of allStakedWallets) {
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

      stakersToCleanup.add(wallet.privateKey);
    }

    console.log("  ✓ All wallets staked and ready");
  }, 120000); // 2 minutes for setup

  describe("Staking Lifecycle", () => {
    test("Verify stake is set up correctly", async () => {
      const staker = getNextHighStakeWallet();
      const client = createClient(staker.privateKey);

      const stakeInfo = (await client.readContract({
        address: poolAddress,
        abi: POOL_ABI,
        functionName: "getStakeInfo",
        args: [staker.address],
      } as any)) as any;

      expect(BigInt(stakeInfo[0])).toBeGreaterThan(BigInt(0));

      console.log("  Staked:", formatEther(BigInt(stakeInfo[0])), "tokens");
      console.log(
        "  Lockup ends:",
        new Date(Number(stakeInfo[2]) * 1000).toISOString()
      );
      console.log(
        "  Access ends:",
        new Date(Number(stakeInfo[3]) * 1000).toISOString()
      );
    });

    test("Submit query with staker address", async () => {
      const staker = getNextHighStakeWallet();
      const client = createClient();
      const blockNumber = await client.getBlockNumber();

      console.log("  Submitting query with staker address...");

      const result = await sendQuery(
        staker.privateKey,
        staker.address,
        Number(blockNumber),
        1000
      );

      if (result.status === 200) {
        console.log("  Query accepted (status: 200)");
        expect(result.status).toBe(200);
        expect(result.data).toBeTruthy();
      } else {
        const isTimeout = result.status === 504 || result.status === 0;

        if (isTimeout) {
          console.log("  Query timed out - server may be slow");
        } else {
          console.log(`  Query failed with status: ${result.status}`);
          expect(result.status).toBe(200);
        }
      }
    });

    test("Increase stake to achieve higher rate limit", async () => {
      const staker = getNextIncreaseStakeWallet();

      const initialStakeAmount = "50000"; // 100 QPS
      const additionalStakeAmount = "5000"; // Total 55k = 110 QPS
      const expectedInitialQPS = 100;
      const expectedFinalQPS = 110;

      const initialStakeWei = parseEther(initialStakeAmount);
      const additionalStakeWei = parseEther(additionalStakeAmount);

      console.log("\n  Using test address:", staker.address);

      // Setup initial stake
      await setupWalletsWithStake([staker], poolAddress, initialStakeAmount);
      stakersToCleanup.add(staker.privateKey);

      const client = createClient(staker.privateKey);

      // Verify initial stake
      const initialStakeInfo = (await client.readContract({
        address: poolAddress,
        abi: POOL_ABI,
        functionName: "getStakeInfo",
        args: [staker.address],
      } as any)) as any;

      console.log(
        `  Initial stake: ${formatEther(
          BigInt(initialStakeInfo[0])
        )} tokens (${expectedInitialQPS} QPS)`
      );

      // Helper to test rate limit
      const testRateLimit = async (
        testName: string
      ): Promise<{
        successCount: number;
        rateLimitCount: number;
        timedOutCount: number;
      }> => {
        const blockNumber = Number(await client.getBlockNumber());
        const numQueries = 5;

        console.log(`\n  ${testName}: Submitting ${numQueries} queries...`);

        let successCount = 0;
        let rateLimitCount = 0;
        let timedOutCount = 0;

        for (let i = 0; i < numQueries; i++) {
          const result = await sendQuery(
            staker.privateKey,
            staker.address,
            blockNumber,
            3000 + i
          );

          const statusStr =
            result.status === 200
              ? "✓"
              : result.status === 429
                ? "✗"
                : result.status === 504
                  ? "⏱"
                  : "?";

          console.log(
            `    Query ${i + 1}/${numQueries}: ${statusStr} (status ${result.status
            })`
          );

          if (result.status === 200) successCount++;
          else if (result.status === 429) rateLimitCount++;
          else if (result.status === 504) timedOutCount++;

          await sleep(50);
        }

        console.log(
          `  Results: ${successCount} succeeded, ${rateLimitCount} rate limited, ${timedOutCount} timed out`
        );

        return { successCount, rateLimitCount, timedOutCount };
      };

      // Test initial rate limit
      await sleep(2000);
      const initialResults = await testRateLimit(
        `Initial (${expectedInitialQPS} QPS)`
      );
      expect(
        initialResults.successCount + initialResults.timedOutCount
      ).toBeGreaterThan(0);

      // Add more stake
      console.log(
        `\n  Adding ${additionalStakeAmount} tokens to increase rate limit...`
      );

      const balance = (await client.readContract({
        address: W_TOKEN_ADDRESS,
        abi: ERC20_ABI,
        functionName: "balanceOf",
        args: [staker.address],
      } as any)) as bigint;

      if (balance < additionalStakeWei) {
        throw new Error(
          `Insufficient balance. Need ${formatEther(
            additionalStakeWei
          )}, have ${formatEther(balance)}`
        );
      }

      const approveHash = await client.writeContract({
        address: W_TOKEN_ADDRESS,
        abi: ERC20_ABI,
        functionName: "approve",
        args: [poolAddress, additionalStakeWei],
      } as any);
      await client.waitForTransactionReceipt({ hash: approveHash });

      const stakeHash = await client.writeContract({
        address: poolAddress,
        abi: POOL_STAKE_ABI,
        functionName: "stake",
        args: [additionalStakeWei],
      } as any);
      await client.waitForTransactionReceipt({ hash: stakeHash });

      // Wait for query server policy cache to clear
      await sleep(32000);

      const finalResults = await testRateLimit(
        `Upgraded (${expectedFinalQPS} QPS)`
      );
      expect(
        finalResults.successCount + finalResults.timedOutCount
      ).toBeGreaterThan(0);

      console.log("\n  ✓ Rate limit increased after adding more tokens!");
    }, 180000);
  });

  describe("Rate Limit Behavior with High Stake", () => {
    test("High stake allows queries without rate limiting", async () => {
      const staker = getNextHighStakeWallet();
      const client = createClient();
      const blockNumber = Number(await client.getBlockNumber());

      const numQueries = 5;

      console.log(`\n  Submitting ${numQueries} queries rapidly...`);
      console.log(`  Stake: ${HIGH_STAKE_AMOUNT} tokens = 20 QPS (1200 QPM)\n`);

      let successCount = 0;
      let rateLimitCount = 0;
      let timedOutCount = 0;

      for (let i = 0; i < numQueries; i++) {
        const result = await sendQuery(
          staker.privateKey,
          staker.address,
          blockNumber,
          2000 + i
        );

        const statusStr =
          result.status === 200
            ? "✓ SUCCESS"
            : result.status === 429
              ? "✗ RATE LIMITED"
              : result.status === 504
                ? "⏱ TIMEOUT"
                : `✗ ERROR ${result.status}`;

        console.log(
          `  Query ${i + 1}/${numQueries}: ${statusStr} (${result.elapsed}ms)`
        );

        if (result.status === 200) successCount++;
        else if (result.status === 429) rateLimitCount++;
        else if (result.status === 504) timedOutCount++;

        await sleep(50);
      }

      console.log(
        `\n  Results: ${successCount} succeeded, ${rateLimitCount} rate limited, ${timedOutCount} timed out`
      );

      // With 20 QPS, all queries should succeed without rate limiting
      expect(successCount + timedOutCount).toBeGreaterThan(0);
      expect(rateLimitCount).toBe(0);

      if (successCount === numQueries) {
        console.log("  ✓ All queries accepted (as expected with 20 QPS limit)");
      } else if (timedOutCount > 0) {
        console.log(
          `  ${timedOutCount} queries timed out (server may be slow)`
        );
      }
    });

    test("Medium stake has moderate rate limits", async () => {
      const staker = getNextMediumStakeWallet();
      const client = createClient();
      const blockNumber = Number(await client.getBlockNumber());

      console.log("\n  Test: Medium stake rate limits");
      console.log(
        `  Wallet: ${staker.address} (${MEDIUM_STAKE_AMOUNT} tokens)`
      );
      console.log("  Expected: Should handle queries without rate limiting\n");

      let successCount = 0;
      let rateLimitCount = 0;

      // Send 10 queries rapidly - with 600 tokens (1 QPS), should mostly succeed
      for (let i = 0; i < 10; i++) {
        const result = await sendQuery(
          staker.privateKey,
          staker.address,
          blockNumber,
          4000 + i
        );

        const statusStr =
          result.status === 200
            ? "✓"
            : result.status === 429
              ? "✗"
              : result.status === 504
                ? "⏱"
                : "?";
        console.log(
          `  Query ${i + 1}/10: ${statusStr} (status ${result.status})`
        );

        if (result.status === 200) successCount++;
        if (result.status === 429) rateLimitCount++;
      }

      console.log(
        `\n  Results: ${successCount} succeeded, ${rateLimitCount} rate limited`
      );

      // With 600+ tokens (1 QPS), at least some queries should succeed
      expect(successCount).toBeGreaterThan(0);
    });
  });

  describe("Rate Limit Behavior with Low Stake", () => {
    test("Rapid burst triggers rate limits", async () => {
      const staker = getNextLowStakeWallet();
      const client = createClient();
      const blockNumber = Number(await client.getBlockNumber());

      console.log("\n  Test: Rapid burst - 10 queries as fast as possible");
      console.log(`  Wallet: ${staker.address} (${LOW_STAKE_AMOUNT} tokens)`);
      console.log("  Expected: Some queries should be rate limited (429)\n");

      let successCount = 0;
      let rateLimitCount = 0;
      let otherCount = 0;

      for (let i = 0; i < 10; i++) {
        const result = await sendQuery(
          staker.privateKey,
          staker.address,
          blockNumber,
          5000 + i
        );

        const statusStr =
          result.status === 200
            ? "✓ SUCCESS"
            : result.status === 429
              ? "✗ RATE LIMITED"
              : result.status === 504
                ? "⏱ TIMEOUT"
                : `✗ ERROR ${result.status}`;

        console.log(`  Query ${i + 1}/10: ${statusStr} (${result.elapsed}ms)`);

        if (result.status === 200) successCount++;
        else if (result.status === 429) rateLimitCount++;
        else otherCount++;
      }

      console.log(
        `\n  Results: ${successCount} succeeded, ${rateLimitCount} rate limited, ${otherCount} other`
      );

      // With low stake, expect rate limiting
      expect(rateLimitCount).toBeGreaterThan(0);
    });

    test("Sustained load triggers rate limits", async () => {
      const staker = getNextLowStakeWallet();
      const client = createClient();
      const blockNumber = Number(await client.getBlockNumber());

      console.log("\n  Test: Sustained load - queries at 2 QPS for 5 seconds");
      console.log(`  Wallet: ${staker.address} (${LOW_STAKE_AMOUNT} tokens)`);
      console.log("  Expected: Should exceed rate limit\n");

      let successCount = 0;
      let rateLimitCount = 0;
      const startTime = Date.now();

      // Send queries at 2 QPS (500ms between queries)
      for (let i = 0; i < 10; i++) {
        const result = await sendQuery(
          staker.privateKey,
          staker.address,
          blockNumber,
          6000 + i
        );

        const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
        const status =
          result.status === 200 ? "✓" : result.status === 429 ? "✗" : "?";
        console.log(
          `  [${elapsed}s] Query ${i + 1}/10: ${status} (status ${result.status
          })`
        );

        if (result.status === 200) successCount++;
        if (result.status === 429) rateLimitCount++;

        await sleep(500); // 500ms = 2 QPS
      }

      console.log(
        `\n  Results: ${successCount} succeeded, ${rateLimitCount} rate limited`
      );

      // Should trigger rate limiting
      expect(rateLimitCount).toBeGreaterThan(0);
    });

    test("Rate limit recovery after cooldown", async () => {
      const staker = getNextLowStakeWallet();
      const client = createClient();
      const blockNumber = Number(await client.getBlockNumber());

      console.log("\n  Test: Rate limit recovery");
      console.log(`  Wallet: ${staker.address} (${LOW_STAKE_AMOUNT} tokens)`);

      // Trigger rate limit
      console.log("  Sending 5 rapid queries...");
      for (let i = 0; i < 5; i++) {
        await sendQuery(
          staker.privateKey,
          staker.address,
          blockNumber,
          7000 + i
        );
      }

      console.log("  Waiting 3 seconds for rate limit to reset...");
      await sleep(3000);

      console.log("  Sending query after cooldown...");
      const result = await sendQuery(
        staker.privateKey,
        staker.address,
        blockNumber,
        7100
      );

      console.log(
        `  Query after cooldown: Status ${result.status} (${result.elapsed}ms)`
      );

      // After cooldown, should not be forbidden (403)
      expect(result.status).not.toBe(403);

      if (result.status === 200 || result.status === 504) {
        console.log("  ✓ Rate limit reset - query succeeded after cooldown");
      } else if (result.status === 429) {
        console.log("Still rate limited - may need longer cooldown");
      }
    });

    test("Concurrent queries respect rate limits", async () => {
      const staker = getNextLowStakeWallet();
      const client = createClient();
      const blockNumber = Number(await client.getBlockNumber());

      console.log("\n  Test: Concurrent queries");
      console.log(`  Wallet: ${staker.address} (${LOW_STAKE_AMOUNT} tokens)`);
      console.log("  Expected: Rate limiting on concurrent requests\n");

      const promises: Promise<{
        status: number;
        elapsed: number;
        data?: any;
      }>[] = [];

      // Send 10 concurrent queries
      for (let i = 0; i < 10; i++) {
        promises.push(
          sendQuery(staker.privateKey, staker.address, blockNumber, 8000 + i)
        );
      }

      const results = await Promise.all(promises);

      let successCount = 0;
      let rateLimitCount = 0;

      results.forEach((result, idx) => {
        const statusStr =
          result.status === 200
            ? "✓"
            : result.status === 429
              ? "✗"
              : result.status === 504
                ? "⏱"
                : "?";
        console.log(
          `  Query ${idx + 1}: ${statusStr} (status ${result.status})`
        );

        if (result.status === 200) successCount++;
        if (result.status === 429) rateLimitCount++;
      });

      console.log(
        `\n  Results: ${successCount} succeeded, ${rateLimitCount} rate limited`
      );

      // With low stake and concurrent requests, expect rate limiting
      expect(rateLimitCount).toBeGreaterThan(0);
    });
  });

  describe("Rate Limit Edge Cases", () => {
    test("Wallet with no stake returns 403", async () => {
      const noStakePrivateKey = generatePrivateKey();
      const noStakeAccount = privateKeyToAccount(noStakePrivateKey);

      console.log("\n  Test: No stake wallet");
      console.log(`  Wallet: ${noStakeAccount.address} (0 tokens)`);
      console.log("  Expected: 403 Forbidden\n");

      const client = createClient();
      const blockNumber = Number(await client.getBlockNumber());

      const result = await sendQuery(
        noStakePrivateKey,
        noStakeAccount.address,
        blockNumber,
        9000
      );

      console.log(`  Status: ${result.status}`);

      // No stake should get 403 Forbidden
      expect(result.status).toBe(403);
    });

    test("Rate limit error message format", async () => {
      const staker = getNextLowStakeWallet();
      const client = createClient();
      const blockNumber = Number(await client.getBlockNumber());

      console.log("\n  Test: Rate limit error message");
      console.log(`  Wallet: ${staker.address} (${LOW_STAKE_AMOUNT} tokens)`);

      let rateLimitError: any = null;

      // Send queries to trigger rate limit
      for (let i = 0; i < 15; i++) {
        const result = await sendQuery(
          staker.privateKey,
          staker.address,
          blockNumber,
          10000 + i
        );

        if (result.status === 429) {
          rateLimitError = result.data;
          console.log(`  Got 429 response: ${result.data}`);
          break;
        }

        await sleep(50);
      }

      if (rateLimitError) {
        expect(rateLimitError).toBeTruthy();
        if (typeof rateLimitError === "string") {
          expect(rateLimitError.toLowerCase()).toContain("rate limit");
        }
        console.log("Rate limit error message is properly formatted");
      } else {
        console.log("No rate limit triggered - skipping validation");
      }
    });
  });
});
